package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Programa all-in-one para ERC-20 (comentários em português, estilo aula).
// Suporta: deploy (usa bytecode do artefato quando disponível), balance, transfer,
// approve, allowance, mint e burn.
//
// Objetivo didático:
// - Explicar passo a passo o que cada parte faz (conectar ao nó, montar chamadas, assinar tx).
// - Usar analogias simples para fixar conceitos (caixa registradora, cheque bancário, procuração).
//
// Analogia (visão geral): pense no contrato ERC-20 como uma 'caixa registradora' compartilhada.
// - `balanceOf`: é como olhar a linha do seu nome no livro-caixa.
// - `transfer`: é como passar fichas de uma carteira para outra.
// - `approve` + `transferFrom`: é como dar uma procuração para alguém gastar em seu nome.
// - `mint`/`burn`: imprimir ou queimar fichas no sistema.
//
// Observação prática: este programa tenta automaticamente carregar o ABI/bytecode gerado
// pelo Foundry em `out/MyToken.sol/MyToken.json` (flag `-artifact`). Se não existir, o
// deploy exige que você forneça bytecode manualmente (ou use o `forge script`, recomendado).

var erc20ABI = `
[{"type":"constructor","inputs":[{"name":"initialSupply","type":"uint256","internalType":"uint256"}],"stateMutability":"nonpayable"},{"type":"function","name":"allowance","inputs":[{"name":"","type":"address","internalType":"address"},{"name":"","type":"address","internalType":"address"}],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"approve","inputs":[{"name":"spender","type":"address","internalType":"address"},{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"nonpayable"},{"type":"function","name":"balanceOf","inputs":[{"name":"","type":"address","internalType":"address"}],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"burn","inputs":[{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"nonpayable"},{"type":"function","name":"decimals","inputs":[],"outputs":[{"name":"","type":"uint8","internalType":"uint8"}],"stateMutability":"view"},{"type":"function","name":"mint","inputs":[{"name":"to","type":"address","internalType":"address"},{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"nonpayable"},{"type":"function","name":"name","inputs":[],"outputs":[{"name":"","type":"string","internalType":"string"}],"stateMutability":"view"},{"type":"function","name":"owner","inputs":[],"outputs":[{"name":"","type":"address","internalType":"address"}],"stateMutability":"view"},{"type":"function","name":"symbol","inputs":[],"outputs":[{"name":"","type":"string","internalType":"string"}],"stateMutability":"view"},{"type":"function","name":"totalSupply","inputs":[],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"transfer","inputs":[{"name":"to","type":"address","internalType":"address"},{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"nonpayable"},{"type":"function","name":"transferFrom","inputs":[{"name":"from","type":"address","internalType":"address"},{"name":"to","type":"address","internalType":"address"},{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"nonpayable"},{"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true,"internalType":"address"},{"name":"spender","type":"address","indexed":true,"internalType":"address"},{"name":"value","type":"uint256","indexed":false,"internalType":"uint256"}],"anonymous":false},{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true,"internalType":"address"},{"name":"to","type":"address","indexed":true,"internalType":"address"},{"name":"value","type":"uint256","indexed":false,"internalType":"uint256"}],"anonymous":false}]
`

// Atenção: substitua por bytecode completo se desejar deploy via Go.
var erc20BinPlaceholder = "0x6080604052"

// loadArtifact tenta carregar ABI e bytecode do artefato do Foundry.
func loadArtifact(path string) (string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	var doc struct {
		ABI              json.RawMessage `json:"abi"`
		DeployedBytecode struct {
			Object string `json:"object"`
		} `json:"deployedBytecode"`
		Bytecode struct {
			Object string `json:"object"`
		} `json:"bytecode"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return "", "", err
	}
	abiStr := ""
	if len(doc.ABI) > 0 {
		abiStr = string(doc.ABI)
	}
	bin := ""
	if doc.DeployedBytecode.Object != "" {
		bin = doc.DeployedBytecode.Object
	} else if doc.Bytecode.Object != "" {
		bin = doc.Bytecode.Object
	}
	return abiStr, bin, nil
}

// Analogia: carregar o artefato é como abrir uma gaveta onde o compilador deixou
// a receita (ABI) e o pacote pronto para deploy (bytecode). Se a gaveta existir,
// usamos essas informações — assim não precisamos copiar/colar bytecode manualmente.

func main() {
	action := flag.String("action", "", "acao: deploy,balance,transfer,approve,allowance,mint,burn")
	rpc := flag.String("rpc", "http://localhost:8545", "RPC URL")
	pk := flag.String("pk", "", "private key hex (0x...) para deploy/txs")
	contract := flag.String("contract", "", "endereço do contrato (para ações após deploy)")
	account := flag.String("account", "", "endereço para checar saldo/allowance")
	to := flag.String("to", "", "destinatário para transfer/approve/mint")
	amount := flag.String("amount", "0", "quantidade em tokens inteiros (será multiplicado por 1e18)")
	supply := flag.String("supply", "1000000", "supply inicial para deploy (tokens inteiros)")
	artifact := flag.String("artifact", "out/MyToken.sol/MyToken.json", "caminho para o artefato JSON do Foundry (opcional)")
	flag.Parse()

	if *action == "" {
		log.Fatalf("-action é obrigatório")
	}

	client, err := ethclient.Dial(*rpc)
	if err != nil {
		log.Fatalf("falha ao conectar RPC: %v", err)
	}
	defer client.Close()

	// Se existir o artefato do Foundry, carregue ABI e bytecode automaticamente.
	if abiStr, bin, err := loadArtifact(*artifact); err == nil {
		if abiStr != "" {
			erc20ABI = abiStr
		}
		if bin != "" {
			erc20BinPlaceholder = bin
		}
		fmt.Println("Usando ABI/bytecode carregados de out/MyToken.sol/MyToken.json")
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		log.Fatalf("ABI inválida: %v", err)
	}

	ctx := context.Background()

	switch *action {
	case "deploy":
		if *pk == "" {
			log.Fatalf("-pk requerido para deploy")
		}
		s, ok := new(big.Int).SetString(*supply, 10)
		if !ok {
			log.Fatalf("supply inválido")
		}
		s = new(big.Int).Mul(s, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		addr, tx := deploy(ctx, client, parsedABI, erc20BinPlaceholder, *pk, s)
		fmt.Println("Contrato deployado em:", addr.Hex())
		fmt.Println("Tx hash:", tx.Hash().Hex())

	case "balance":
		if *contract == "" || *account == "" {
			log.Fatalf("-contract e -account requeridos")
		}
		bal := callBalance(ctx, client, parsedABI, common.HexToAddress(*contract), common.HexToAddress(*account))
		fmt.Printf("Saldo (raw): %s\n", bal.String())

	case "transfer":
		if *pk == "" || *contract == "" || *to == "" {
			log.Fatalf("-pk -contract -to requeridos")
		}
		amt := parseTokenAmount(*amount)
		tx := transact(ctx, client, parsedABI, common.HexToAddress(*contract), *pk, "transfer", common.HexToAddress(*to), amt)
		fmt.Println("Tx hash:", tx.Hash().Hex())

	case "mint":
		if *pk == "" || *contract == "" || *to == "" {
			log.Fatalf("-pk -contract -to requeridos")
		}
		amt := parseTokenAmount(*amount)
		tx := transact(ctx, client, parsedABI, common.HexToAddress(*contract), *pk, "mint", common.HexToAddress(*to), amt)
		fmt.Println("Tx hash:", tx.Hash().Hex())

	case "burn":
		if *pk == "" || *contract == "" {
			log.Fatalf("-pk -contract requeridos")
		}
		amt := parseTokenAmount(*amount)
		tx := transact(ctx, client, parsedABI, common.HexToAddress(*contract), *pk, "burn", amt)
		fmt.Println("Tx hash:", tx.Hash().Hex())

	case "approve":
		if *pk == "" || *contract == "" || *to == "" {
			log.Fatalf("-pk -contract -to requeridos")
		}
		amt := parseTokenAmount(*amount)
		tx := transact(ctx, client, parsedABI, common.HexToAddress(*contract), *pk, "approve", common.HexToAddress(*to), amt)
		fmt.Println("Tx hash:", tx.Hash().Hex())

	case "allowance":
		if *contract == "" || *account == "" || *to == "" {
			log.Fatalf("-contract -account -to requeridos")
		}
		alw := callAllowance(ctx, client, parsedABI, common.HexToAddress(*contract), common.HexToAddress(*account), common.HexToAddress(*to))
		fmt.Println("Allowance (raw):", alw.String())

	default:
		log.Fatalf("acao desconhecida: %s", *action)
	}
}

// parseTokenAmount converte um valor inteiro (ex: "100") para a unidade interna (wei do token)
func parseTokenAmount(s string) *big.Int {
	// Analogia: converter unidades inteiras de token para a unidade interna (como transformar
	// "reais" em "centavos"). Aqui tokens inteiros -> multiplicamos por 1e18.
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		log.Fatalf("amount inválido")
	}
	return new(big.Int).Mul(v, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

// newAuth cria um TransactOpts com nonce e gas estimado usando a chave privada
func newAuth(ctx context.Context, client *ethclient.Client, pkHex string) (*bind.TransactOpts, error) {
	// Analogia: aqui transformamos sua chave privada em um 'cartão de assinatura'
	// que será usado para assinar transações. Também populamos o nonce e sugerimos
	// um gas price para a transação — similar a preencher os dados do envelope antes
	// de enviar um cheque ao banco.
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
	if err != nil {
		return nil, err
	}
	pub := pk.Public()
	pubKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("chave publica invalida")
	}
	from := crypto.PubkeyToAddress(*pubKey)
	chainID := big.NewInt(31337)
	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return nil, err
	}
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	auth.Nonce = new(big.Int).SetUint64(nonce)
	if gp, err := client.SuggestGasPrice(ctx); err == nil {
		auth.GasPrice = gp
	}
	auth.GasLimit = uint64(500000)
	auth.Context = ctx
	return auth, nil
}

// deploy envia o bytecode para a rede e retorna o endereço do contrato
func deploy(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, binPlaceholder string, pkHex string, initialSupply *big.Int) (common.Address, *types.Transaction) {
	// Nota didática: aqui usamos um placeholder para bytecode. Para deploy real, copie o
	// deployed bytecode do artefato do compilador (ex: out/MyToken.sol/MyToken.json -> deployedBytecode.object)
	// Analogia: deploy é como comprar e colocar uma nova máquina numa praça pública.
	// O bytecode é o manual/arquivo binário que a máquina precisa para funcionar.
	// Se usamos o artefato do Foundry, já temos esse manual pronto.
	auth, err := newAuth(ctx, client, pkHex)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	bin := common.FromHex(binPlaceholder)
	addr, tx, _, err := bind.DeployContract(auth, parsedABI, bin, client, initialSupply)
	if err != nil {
		log.Fatalf("deploy falhou: %v", err)
	}
	waitTx(ctx, client, tx)
	return addr, tx
}

// waitTx aguarda receipt da transação (simples)
func waitTx(ctx context.Context, client *ethclient.Client, tx *types.Transaction) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		receipt, err := client.TransactionReceipt(timeoutCtx, tx.Hash())
		if err == nil && receipt != nil {
			return
		}
		select {
		case <-timeoutCtx.Done():
			log.Fatalf("timeout esperando tx")
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// callBalance usa CallContract + ABI.Unpack para obter balanceOf
func callBalance(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, contract common.Address, who common.Address) *big.Int {
	// Analogia: chamada view (call) é como pedir ao caixa registradora para te mostrar
	// um número no livro; não altera o estado, só lê.
	data, err := parsedABI.Pack("balanceOf", who)
	if err != nil {
		log.Fatalf("pack balanceOf: %v", err)
	}
	msg := ethereum.CallMsg{To: &contract, Data: data}
	res, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		log.Fatalf("call contract: %v", err)
	}
	vals, err := parsedABI.Unpack("balanceOf", res)
	if err != nil {
		log.Fatalf("unpack balanceOf: %v", err)
	}
	if len(vals) == 0 {
		return big.NewInt(0)
	}
	v, ok := vals[0].(*big.Int)
	if !ok {
		log.Fatalf("tipo inesperado para balanceOf")
	}
	return v
}

// callAllowance semelhante a callBalance
func callAllowance(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, contract common.Address, owner common.Address, spender common.Address) *big.Int {
	data, err := parsedABI.Pack("allowance", owner, spender)
	if err != nil {
		log.Fatalf("pack allowance: %v", err)
	}
	msg := ethereum.CallMsg{To: &contract, Data: data}
	res, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		log.Fatalf("call contract: %v", err)
	}
	vals, err := parsedABI.Unpack("allowance", res)
	if err != nil {
		log.Fatalf("unpack allowance: %v", err)
	}
	if len(vals) == 0 {
		return big.NewInt(0)
	}
	v, ok := vals[0].(*big.Int)
	if !ok {
		log.Fatalf("tipo inesperado para allowance")
	}
	return v
}

// transact executa uma chamada de transação (transfer, approve, mint, burn)
func transact(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, contract common.Address, pkHex string, method string, params ...interface{}) *types.Transaction {
	// Analogia: transação é enviar um cheque assinado ao banco para mover fichas.
	// Aqui criamos um TransactOpts (auth) com a chave, empacotamos a chamada e enviamos.
	auth, err := newAuth(ctx, client, pkHex)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	bound := bind.NewBoundContract(contract, parsedABI, client, client, client)
	tx, err := bound.Transact(auth, method, params...)
	if err != nil {
		log.Fatalf("transact %s: %v", method, err)
	}
	waitTx(ctx, client, tx)
	return tx
}
