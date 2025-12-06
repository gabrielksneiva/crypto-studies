package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"time"

	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// ========== PASSO 0: CONFIGURAÇÃO ==========
	// Analogia: Você precisa conectar a um nó Ethereum (como Infura, Alchemy ou um nó local).
	// Informações necessárias:
	//   - RPC endpoint: endereço da API do nó Ethereum
	//   - Chainid: qual rede (1=mainnet, 5=goerli, 11155111=sepolia, 31337=anvil local)
	//   - Conta: qual endereço você quer usar para assinar a transação

	rpcEndpoint := flag.String("rpc", "http://127.0.0.1:8545", "RPC endpoint do nó Ethereum")
	chainID := flag.Int64("chain", 31337, "Chain ID (31337=anvil, 5=goerli, 1=mainnet)")
	mnemonic := flag.String("mnemonic", "test test test test test test test test test test test junk", "Frase mnemônica BIP39")
	accountIndex := flag.Int("account", 0, "Índice da conta (m/44'/60'/0'/0/N)")

	// ========== PASSO 1: INFORMAÇÕES DA TRANSAÇÃO ==========
	// Analogia: Você quer fazer uma transferência no Ethereum. Precisa de:
	//   - To: endereço de destino (quem vai receber)
	//   - Value: quanto ETH enviar (em wei, 1 ETH = 10^18 wei)
	//   - Gas: combustível para pagar a transação (análogo a taxa)
	//   - Nonce: número sequencial (para prevenir replay attacks)

	toAddr := flag.String("to", "0x0000000000000000000000000000000000000000", "Endereço de destino (0x...)")
	valueETH := flag.Float64("value", 0.1, "Quantidade de ETH a enviar")
	gasLimit := flag.Int64("gas", 21000, "Gas limit (21000 para transfer simples)")
	gasPrice := flag.Int64("gasprice", 0, "Gas price em wei (0 = auto-descoberta)")
	nonce := flag.Int64("nonce", -1, "Nonce da transação (-1 = auto-descoberta)")

	flag.Parse()

	// ========== VALIDAÇÃO ==========
	if *toAddr == "" || *valueETH < 0 {
		panic("preencha to e value")
	}

	fmt.Println("========== PASSO 0: CONFIGURAÇÃO ==========")
	fmt.Printf("RPC Endpoint: %s\n", *rpcEndpoint)
	fmt.Printf("Chain ID: %d\n", *chainID)
	fmt.Printf("Mnemonic: %s\n", *mnemonic)
	fmt.Printf("Account index: %d\n\n", *accountIndex)

	// ========== PASSO 1: GERAR CHAVE PRIVADA DA MNEMÔNICA ==========
	// Analogia: Você converte sua frase mnemônica em uma chave privada usável.
	// Processo:
	//   1. Converter mnemônico em seed (BIP39)
	//   2. Derivar chave usando caminho BIP44 Ethereum
	//   3. Extrair chave privada de 32 bytes

	fmt.Println("========== PASSO 1: DERIVAÇÃO BIP44 ETHEREUM ==========")

	seed := bip39.NewSeed(*mnemonic, "")
	net := &chaincfg.TestNet3Params

	master, err := hd.NewMaster(seed, net)
	if err != nil {
		log.Fatalf("erro ao criar master key: %v", err)
	}

	// Derivar caminho: m/44'/60'/0'/0/accountIndex
	purpose, _ := master.Derive(hd.HardenedKeyStart + 44)
	coinType, _ := purpose.Derive(hd.HardenedKeyStart + 60)
	account, _ := coinType.Derive(hd.HardenedKeyStart + 0)
	change, _ := account.Derive(0)
	addressKey, err := change.Derive(uint32(*accountIndex))
	if err != nil {
		log.Fatalf("erro na derivação: %v", err)
	}

	fmt.Printf("✓ Derivado: m/44'/60'/0'/0/%d\n", *accountIndex)
	fmt.Println()

	// ========== PASSO 2: EXTRAIR CHAVE PRIVADA ==========
	// Analogia: A chave privada é o segredo que permite assinar a transação.
	// Sem ela, qualquer pessoa poderia roubar seus fundos.

	fmt.Println("========== PASSO 2: CHAVE PRIVADA ==========")

	privKeyECDSA, err := addressKey.ECPrivKey()
	if err != nil {
		log.Fatalf("erro ao extrair chave privada: %v", err)
	}

	privKeyBytes := privKeyECDSA.Serialize()
	privKeyHex := hex.EncodeToString(privKeyBytes)

	fmt.Printf("Chave privada: %s\n", privKeyHex)
	fmt.Println("⚠️  NUNCA compartilhe isto! Qualquer pessoa com isto pode roubar seus ETH")
	fmt.Println()

	// Converter para chave privada ECDSA do go-ethereum
	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		log.Fatalf("erro ao converter para ECDSA: %v", err)
	}

	// Derivar endereço público
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("erro ao converter public key")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Printf("Seu endereço (from): %s\n\n", fromAddress.Hex())

	// ========== PASSO 3: CONECTAR AO NÓ ETHEREUM ==========
	// Analogia: É como fazer login no app do banco para poder enviar dinheiro.
	// Você precisa de um nó Ethereum rodando (Geth, Anvil, Hardhat, Infura, etc.)

	fmt.Println("========== PASSO 3: CONECTAR AO NÓ ETHEREUM ==========")
	fmt.Printf("Conectando em: %s\n", *rpcEndpoint)

	client, err := ethclient.Dial(*rpcEndpoint)
	if err != nil {
		log.Fatalf("erro ao conectar no RPC: %v\nCertifique-se de que um nó está rodando (ex: anvil, geth, etc.)", err)
	}
	defer client.Close()

	fmt.Println("✓ Conectado com sucesso!")
	fmt.Println()

	// Auto-descoberta: buscar nonce automaticamente
	ctx := context.Background()
	if *nonce == -1 {
		nonceVal, err := client.PendingNonceAt(ctx, fromAddress)
		if err != nil {
			log.Fatalf("erro ao buscar nonce: %v", err)
		}
		*nonce = int64(nonceVal)
		fmt.Printf("✓ Nonce obtido automaticamente: %d\n", *nonce)
	}

	// Auto-descoberta: buscar gas price automaticamente se não informado
	if *gasPrice == 0 {
		gasPriceVal, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("erro ao buscar gas price: %v", err)
		}
		*gasPrice = gasPriceVal.Int64()
		fmt.Printf("✓ Gas price obtido automaticamente: %d wei (%.2f Gwei)\n", *gasPrice, float64(*gasPrice)/1e9)
	}
	fmt.Println()

	// ========== PASSO 4: INFORMAÇÕES DA TRANSAÇÃO ==========
	// Analogia: Você está montando um cheque bancário com:
	//   - Para: quem vai receber
	//   - Valor: quanto enviar
	//   - Taxa: quanto pagar de combustível
	//   - Nonce: número sequencial para evitar duplicação

	fmt.Println("========== PASSO 4: DADOS DA TRANSAÇÃO ==========")

	valueWei := new(big.Float).Mul(big.NewFloat(*valueETH), big.NewFloat(1e18))
	valueWeiBig := new(big.Int)
	valueWei.Int(valueWeiBig)

	toAddress := common.HexToAddress(*toAddr)

	fmt.Printf("De: %s\n", fromAddress.Hex())
	fmt.Printf("Para: %s\n", toAddress.Hex())
	fmt.Printf("Valor: %.10f ETH (%s wei)\n", *valueETH, valueWeiBig.String())
	fmt.Printf("Gas Limit: %d\n", *gasLimit)
	fmt.Printf("Gas Price: %d wei (%.9f Gwei)\n", *gasPrice, float64(*gasPrice)/1e9)
	fmt.Printf("Nonce: %d\n", *nonce)

	totalCost := new(big.Int).Mul(big.NewInt(*gasLimit), big.NewInt(*gasPrice))
	totalCostWithValue := new(big.Int).Add(totalCost, valueWeiBig)

	fmt.Printf("Taxa (Gas Limit × Gas Price): %s wei (%.9f ETH)\n", totalCost.String(), float64(totalCost.Int64())/1e18)
	fmt.Printf("Total (valor + taxa): %s wei (%.9f ETH)\n\n", totalCostWithValue.String(), float64(totalCostWithValue.Int64())/1e18)

	// ========== PASSO 5: MONTAR A TRANSAÇÃO ==========
	// Analogia: É como montar um envelope com os dados da transação.
	// No Ethereum, a transação é codificada em RLP (Recursive Length Prefix).
	//
	// Campos:
	//   - nonce: número sequencial
	//   - gasPrice: preço por unidade de gas
	//   - gasLimit: quantidade máxima de gas
	//   - to: endereço de destino
	//   - value: quantidade de ETH
	//   - data: dados da transação (vazio para transfer simples)

	fmt.Println("========== PASSO 5: MONTANDO A TRANSAÇÃO ==========")

	tx := types.NewTransaction(
		uint64(*nonce),
		toAddress,
		valueWeiBig,
		uint64(*gasLimit),
		big.NewInt(*gasPrice),
		nil, // data vazio para transfer simples
	)

	fmt.Println("Transação criada (não assinada):")
	fmt.Printf("  Hash (provisório): %s\n", tx.Hash().Hex())
	fmt.Printf("  Nonce: %d\n", tx.Nonce())
	fmt.Printf("  To: %s\n", tx.To().Hex())
	fmt.Printf("  Value: %s wei\n", tx.Value().String())
	fmt.Printf("  Gas: %d\n", tx.Gas())
	fmt.Printf("  Gas Price: %s wei\n", tx.GasPrice().String())
	fmt.Println()

	// ========== PASSO 6: ASSINAR A TRANSAÇÃO ==========
	// Analogia: É como você assinar um cheque com sua assinatura única.
	// Quem receber o cheque sabe que foi você quem assinou.
	//
	// Processo (KECCAK256 + ECDSA):
	//   1. Codificar a transação em RLP
	//   2. Fazer KECCAK256 do RLP
	//   3. Assinar o hash com ECDSA (chave privada)
	//   4. Resultado: v, r, s (componentes da assinatura)

	fmt.Println("========== PASSO 6: ASSINANDO A TRANSAÇÃO ==========")

	chainIDBig := big.NewInt(*chainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainIDBig), privKey)
	if err != nil {
		log.Fatalf("erro ao assinar transação: %v", err)
	}

	fmt.Println("✓ Transação assinada com sucesso!")
	fmt.Printf("  Hash (final): %s\n", signedTx.Hash().Hex())

	// Extrair componentes da assinatura
	v, r, s := signedTx.RawSignatureValues()
	fmt.Printf("  v: %d\n", v)
	fmt.Printf("  r: %s\n", r.String())
	fmt.Printf("  s: %s\n", s.String())
	fmt.Println()

	// ========== PASSO 7: BROADCAST (ENVIAR PARA A REDE) ==========
	// Analogia: Você envia o cheque assinado para o banco.
	// O banco verifica a assinatura e processa o pagamento.

	fmt.Println("========== PASSO 7: ENVIANDO PARA A REDE ==========")

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatalf("erro ao enviar transação: %v", err)
	}

	fmt.Println("✓ Transação enviada com sucesso!")
	fmt.Printf("  TX Hash: %s\n\n", signedTx.Hash().Hex())

	// ========== PASSO 8: VERIFICAR NA MEMPOOL ==========
	// Analogia: Verificar se o cheque chegou no banco e está aguardando processamento.

	fmt.Println("========== PASSO 8: VERIFICANDO NA MEMPOOL ==========")

	_, isPending, err := client.TransactionByHash(ctx, signedTx.Hash())
	if err != nil {
		log.Printf("aviso: erro ao verificar mempool: %v", err)
	} else {
		if isPending {
			fmt.Println("✓ Transação encontrada na mempool (aguardando mineração)")
		} else {
			fmt.Println("✓ Transação já foi minerada!")
		}
	}
	fmt.Println()

	// ========== PASSO 9: AGUARDAR CONFIRMAÇÃO ==========
	// Analogia: Esperar o banco processar o cheque e confirmar o pagamento.

	fmt.Println("========== PASSO 9: AGUARDANDO CONFIRMAÇÃO ==========")
	fmt.Println("Aguardando a transação ser minerada em um bloco...")
	fmt.Println("(Pressione Ctrl+C se quiser cancelar a espera)")
	fmt.Println()

	for i := 0; i < 60; i++ { // Tentar por 60 segundos
		time.Sleep(1 * time.Second)

		receipt, err := client.TransactionReceipt(ctx, signedTx.Hash())
		if err == nil {
			// Transação foi minerada!
			fmt.Println("✓ TRANSAÇÃO CONFIRMADA!")
			fmt.Printf("  Bloco: %d\n", receipt.BlockNumber.Uint64())
			fmt.Printf("  Gas usado: %d (%.2f%%)\n", receipt.GasUsed, float64(receipt.GasUsed)/float64(*gasLimit)*100)
			fmt.Printf("  Status: %d (1 = sucesso, 0 = falha)\n", receipt.Status)
			fmt.Printf("  TX Hash: %s\n\n", receipt.TxHash.Hex())

			if receipt.Status == 1 {
				fmt.Println("✅ SUCESSO! Transação executada com sucesso!")
			} else {
				fmt.Println("❌ FALHA! Transação foi revertida (verifique se há saldo suficiente)")
			}

			break
		}

		fmt.Printf(".")
		if (i+1)%10 == 0 {
			fmt.Printf(" %ds\n", i+1)
		}
	}

	fmt.Println()

	// ========== RESUMO ==========
	fmt.Println("========== RESUMO DO FLUXO COMPLETO ==========")
	fmt.Println("1. Mnemônico BIP39")
	fmt.Println("   ↓")
	fmt.Println("2. Derivação BIP44 Ethereum → chave privada")
	fmt.Println("   ↓")
	fmt.Println("3. Conectar ao nó Ethereum (RPC)")
	fmt.Println("   ↓")
	fmt.Println("4. Montar transação (nonce, gasPrice, to, value, etc)")
	fmt.Println("   ↓")
	fmt.Println("5. Assinar transação (ECDSA → v, r, s)")
	fmt.Println("   ↓")
	fmt.Println("6. Enviar via eth_sendRawTransaction")
	fmt.Println("   ↓")
	fmt.Println("7. Aguardar confirmação no blockchain")
	fmt.Println()

	fmt.Println("Diferença Bitcoin vs Ethereum:")
	fmt.Println("  - Bitcoin: UTXO model (cédulas físicas)")
	fmt.Println("  - Ethereum: Account model (saldo na conta)")
	fmt.Println()
	fmt.Println("  - Bitcoin: Derivação BIP84 (SegWit)")
	fmt.Println("  - Ethereum: Derivação BIP44 (coin type 60)")
}
