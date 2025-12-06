package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func main() {
	// ========== PASSO 0: CONFIGURAÇÃO ==========
	// Analogia: Você precisa conectar a um caixa eletrônico (bitcoind node)
	// Informações necessárias:
	//   - username/password: credenciais para autenticar na API
	//   - host:port: endereço da máquina rodando o bitcoind
	//   - wallet: qual carteira usar (pode ter várias abertas)

	rpcUser := flag.String("rpcuser", "admin", "RPC username do bitcoind")
	rpcPass := flag.String("rpcpass", "123456", "RPC password do bitcoind")
	rpcHost := flag.String("rpchost", "127.0.0.1:18443", "host:port do bitcoind regtest")
	rpcWallet := flag.String("wallet", "wallet", "nome do wallet carregado no bitcoind")

	// ========== PASSO 1: INFORMAÇÕES DO UTXO QUE VOCÊ QUER GASTAR ==========
	// UTXO = Unspent Transaction Output
	// Analogia: Imagine que cada "bitcoin" é uma cédula física.
	// Cada cédula tem:
	//   - Um número de série único (txid: qual transação a criou)
	//   - Uma posição (vout: qual saída daquela transação)
	//   - Um valor (amount: quantos satoshis)
	//   - Um dono (scriptPubKey: quem tem direito de gastar)

	utxoTxid := flag.String("utxo_txid", "", "txid do UTXO a gastar")
	utxoVout := flag.Uint("utxo_vout", 0, "vout do UTXO a gastar")
	utxoAmountSat := flag.Int64("utxo_amount_sat", 0, "valor do UTXO em satoshis")
	utxoAddr := flag.String("utxo_addr", "", "endereco (script) do UTXO (ex: bech32 regtest)")
	utxoScriptHex := flag.String("utxo_script", "", "scriptPubKey do UTXO em hex (opcional)")

	// ========== PASSO 2: INFORMAÇÕES DOS OUTPUTS (PARA ONDE O DINHEIRO VAI) ==========
	// Analogia: Você está dividindo a cédula em duas partes:
	//   1. Uma parte vai para o destinatário
	//   2. A outra parte volta para você como troco

	destAddrStr := flag.String("dest", "bcrt1qafjfjrfxnkd9gwtcuahwl7v4ucf6cjg0hs6cz7", "endereco de destino")
	sendSat := flag.Int64("send_sat", 100000000, "quantidade a enviar (satoshis, padrão: 1 BTC)")
	changeAddrStr := flag.String("change", "", "endereco de troco (do seu wallet)")
	feeSat := flag.Int64("fee_sat", 500, "taxa em satoshis")

	// ========== OPÇÕES PÓS-BROADCAST ==========
	// Fluxo completo sempre ativo: criar → assinar → broadcast → verificar → minerar → verificar confirmação

	flag.Parse()
	// ========== VALIDAÇÃO: SÓ PRECISA PASSAR DEST E SEND ==========
	// O resto é preenchido automaticamente pelo programa!
	if *sendSat <= 0 || *destAddrStr == "" {
		panic("preencha send_sat e dest")
	}

	// ========== ESCOLHENDO A REDE: REGTEST ==========
	// Analogia: Existem várias versões do Bitcoin:
	//   - Mainnet: a rede real com satoshis de verdade
	//   - Testnet: rede de testes pública
	//   - Regtest: sua rede privada local (para desenvolvimento)
	// Aqui escolhemos Regtest
	netParams := &chaincfg.RegressionNetParams

	// ========== CONSTRUIR A URL DO RPC ==========
	// Analogia: É como montar o endereço completo:
	// http://127.0.0.1:18443/wallet/wallet
	// Quando você faz requisição HTTP POST aqui, o nó Bitcoin recebe e executa
	rpcURL := fmt.Sprintf("http://%s/wallet/%s", *rpcHost, *rpcWallet)

	// ========== AUTO-DESCOBERTA 1: ENDEREÇO DE TROCO AUTOMÁTICO ==========
	// Se você não passou -change, o programa pede ao nó um novo endereço
	// RPC: getrawchangeaddress
	// Analogia: "Ei nó, gera um novo endereço para eu receber o troco"
	if *changeAddrStr == "" {
		var addrStr string
		// Chama o nó via RPC para gerar novo endereço de troco
		// O parâmetro é uma string simples: "bech32"
		mustRPC(rpcCallResult(rpcURL, *rpcUser, *rpcPass, "getrawchangeaddress", []any{"bech32"}), &addrStr)
		*changeAddrStr = addrStr
		fmt.Printf("Change address gerado automaticamente: %s\n", *changeAddrStr)
	}

	// ========== AUTO-DESCOBERTA 2: UTXO AUTOMÁTICO ==========
	// Se você não passou -utxo_txid, o programa lista seus UTXOs
	// RPC: listunspent
	// Analogia: "Ei nó, quais são as notas que estão na minha carteira?"
	if *utxoTxid == "" {
		// Estrutura para armazenar a resposta do RPC
		var utxos []struct {
			Txid         string  `json:"txid"`         // número de série (identificador único)
			Vout         uint32  `json:"vout"`         // posição dentro da transação
			Address      string  `json:"address"`      // endereço que controla esta nota
			ScriptPubKey string  `json:"scriptPubKey"` // a "chave" em formato hexadecimal
			Amount       float64 `json:"amount"`       // valor em BTC
		}

		// Chamar o nó: lista todos os UTXOs com no mínimo 1 confirmação
		mustRPC(rpcCallResult(rpcURL, *rpcUser, *rpcPass, "listunspent", []any{1, 9999999}), &utxos)

		// Calcular quanto você precisa no mínimo
		// (valor para enviar + taxa que vai pagar)
		need := *sendSat + *feeSat

		// Procurar a primeira nota que seja grande o suficiente
		var picked bool
		for _, u := range utxos {
			// Converter BTC para satoshis (BTC × 100,000,000)
			sats := int64(math.Round(u.Amount * 1e8))

			// Se essa nota cobre o necessário, usar!
			if sats >= need {
				*utxoTxid = u.Txid
				*utxoVout = uint(u.Vout)
				*utxoAmountSat = sats
				*utxoAddr = u.Address
				*utxoScriptHex = u.ScriptPubKey
				picked = true
				fmt.Printf("UTXO selecionado automaticamente:\n  txid: %s\n  vout: %d\n  amount: %d sat\n", *utxoTxid, *utxoVout, *utxoAmountSat)
				break
			}
		}

		if !picked {
			panic("nenhum UTXO cobre send_sat+fee_sat")
		}
	}

	// ========== VALIDAÇÃO FINAL ==========
	if *utxoAmountSat <= 0 || *utxoAddr == "" || *utxoTxid == "" {
		panic("dados do UTXO incompletos")
	}

	// ========== PASSO 3: DECODIFICAR ENDEREÇOS ==========
	// Analogia: Você recebeu endereços em formato legível (bech32: bcrt1q...)
	// Mas para criar a transação, precisa converter para formato binário/hexadecimal
	// É como converter um endereço postal para GPS coordinates

	prevAddr, err := btcutil.DecodeAddress(*utxoAddr, netParams)
	if err != nil {
		panic(err)
	}
	destAddr, err := btcutil.DecodeAddress(*destAddrStr, netParams)
	if err != nil {
		panic(err)
	}
	changeAddr, err := btcutil.DecodeAddress(*changeAddrStr, netParams)
	if err != nil {
		panic(err)
	}

	// ========== PASSO 4: CONVERTER ENDEREÇOS EM SCRIPTS (CADEADOS) ==========
	// Analogia: Cada endereço é uma "fechadura" que tranca o dinheiro.
	// Um script é o código que implementa essa fechadura.
	// Exemplo: um endereço bech32 vira um script P2WPKH (Pay-to-Witness-PubKeyHash)

	var prevPkScript []byte
	if *utxoScriptHex != "" {
		// Se você já tem o script em hex, use-o diretamente
		prevPkScript, err = hex.DecodeString(*utxoScriptHex)
		if err != nil {
			panic(err)
		}
	} else {
		// Caso contrário, derive do endereço
		prevPkScript, err = txscript.PayToAddrScript(prevAddr)
		if err != nil {
			panic(err)
		}
	}

	destScript, err := txscript.PayToAddrScript(destAddr)
	if err != nil {
		panic(err)
	}
	changeScript, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		panic(err)
	}

	// ========== PASSO 5: CALCULAR O TROCO ==========
	// Analogia: Aritmética simples de dinheiro
	// Você tem: 100 satoshis (UTXO)
	// Quer enviar: 50 satoshis
	// Taxa: 10 satoshis
	// Troco: 100 - 50 - 10 = 40 satoshis

	changeValue := *utxoAmountSat - *sendSat - *feeSat
	if changeValue < 0 {
		panic("send_sat + fee_sat > utxo_amount_sat")
	}

	// ========== PASSO 6: MONTAR A TRANSAÇÃO (AINDA SEM ASSINATURA) ==========
	// Analogia: Você está preenchendo um formulário de banco:
	//   - INPUT: "Vou usar este UTXO como entrada"
	//   - OUTPUT 1: "Enviar este valor para este endereço"
	//   - OUTPUT 2: "Enviar o troco para este outro endereço"

	tx := wire.NewMsgTx(wire.TxVersion)

	// ========== 6A: ADICIONAR INPUT (A NOTA QUE VOCÊ QUER GASTAR) ==========
	// Analogia: Você pega uma cédula e coloca na mesa.
	// Você diz: "Vou usar a cédula número [txid] posição [vout]"

	hash, err := chainhash.NewHashFromStr(*utxoTxid)
	if err != nil {
		panic(err)
	}
	outPoint := wire.NewOutPoint(hash, uint32(*utxoVout))
	txIn := wire.NewTxIn(outPoint, nil, nil)
	tx.AddTxIn(txIn)

	// ========== 6B: ADICIONAR OUTPUTS (PARA ONDE O DINHEIRO VAI) ==========
	// Analogia: Preenchendo envelopes:
	//   - Envelope 1: X satoshis para o destinatário (selado com seu endereço)
	//   - Envelope 2: Y satoshis para você (selado com seu endereço de troco)

	tx.AddTxOut(wire.NewTxOut(*sendSat, destScript))
	if changeValue > 0 {
		tx.AddTxOut(wire.NewTxOut(changeValue, changeScript))
	}

	// ========== PASSO 7: SERIALIZAR A TRANSAÇÃO ==========
	// Analogia: Você folding o formulário preenchido e coloca em um envelope.
	// Aqui convertemos a transação de estrutura Go para bytes binários,
	// depois para hexadecimal (texto) para poder transmitir.

	var buf bytes.Buffer
	tx.Serialize(&buf)
	rawTxHex := hex.EncodeToString(buf.Bytes())
	fmt.Println("\n[RAW TX (UNSIGNED)] A transação antes de assinar:")
	fmt.Println(rawTxHex)

	// ========== PASSO 8: ASSINAR (CHAVE PRIVADA) ==========
	// Analogia: Você vai ao notário (nó Bitcoin) e diz:
	// "Assine este documento com a minha chave privada como prova de que sou mesmo eu"
	// RPC: signrawtransactionwithwallet
	// Analogia: "Ei nó, use a chave privada da minha carteira para assinar isto"

	// Preparar os dados do UTXO anterior para o nó
	// (necessário para o nó calcular a assinatura corretamente)
	prevTx := []map[string]any{
		{
			"txid":         *utxoTxid,                        // qual transação criou este UTXO
			"vout":         *utxoVout,                        // qual output daquela transação
			"scriptPubKey": hex.EncodeToString(prevPkScript), // o "cadeado" original
			"amount":       float64(*utxoAmountSat) / 1e8,    // quanto vale (em BTC)
		},
	}

	// Chamar o RPC: signrawtransactionwithwallet
	signParams := []any{rawTxHex, prevTx}
	signRaw := rpcCallResult(rpcURL, *rpcUser, *rpcPass, "signrawtransactionwithwallet", signParams)

	// Decodificar a resposta
	var signResp struct {
		Hex      string `json:"hex"`      // transação assinada em hexadecimal
		Complete bool   `json:"complete"` // conseguiu assinar completamente?
		Errors   []any  `json:"errors"`   // erros se houver
	}
	mustRPC(signRaw, &signResp)

	if !signResp.Complete {
		panic(fmt.Sprintf("assinatura incompleta: %+v", signResp.Errors))
	}

	fmt.Println("\n[RAW TX (SIGNED)] A transação depois de assinar:")
	fmt.Println(signResp.Hex)

	// ========== PASSO 9: BROADCAST (ENVIAR PARA A REDE) ==========
	// Analogia: Você levou a documento assinado ao correio.
	// O nó vai enviar essa transação para todos os outros nós da rede.
	// RPC: sendrawtransaction
	// Analogia: "Ei nó, propague esta transação para a rede inteira"

	sendParams := []any{signResp.Hex}
	sendRaw := rpcCallResult(rpcURL, *rpcUser, *rpcPass, "sendrawtransaction", sendParams)

	// O nó retorna o TXID da transação que foi propagada
	var txidHex string
	mustRPC(sendRaw, &txidHex)

	fmt.Println("\n[BROADCAST SUCESSO]")
	fmt.Printf("Transação enviada para a rede!\nTXID: %s\n", txidHex)
	fmt.Println("\nO que acontece agora:")
	fmt.Println("1. A transação é propagada para todos os nós da rede")
	fmt.Println("2. Entra no mempool (fila de transações aguardando)")
	fmt.Println("3. Mineradores veem e incluem em um bloco")
	fmt.Println("4. Quando o bloco é minado, sua TX tem 1 confirmação")
	fmt.Println("5. Mais blocos adicionados = mais confirmações (maior segurança)")

	// ========== PASSO 10: VERIFICAR A TRANSAÇÃO NO MEMPOOL ==========
	// Analogia: Depois de entregar a carta no correio, você liga para confirmar que chegou.
	// RPC: gettransaction
	// Analogia: "Ei nó, mostra os detalhes dessa transação"
	fmt.Println("\n========== PASSO 10: VERIFICANDO MEMPOOL ==========")
	var txInfo struct {
		TXId          string  `json:"txid"`
		Confirmations int     `json:"confirmations"`
		Time          int64   `json:"time"`
		Timereceived  int64   `json:"timereceived"`
		Fee           float64 `json:"fee"`
	}
	mustRPC(rpcCallResult(rpcURL, *rpcUser, *rpcPass, "gettransaction", []any{txidHex}), &txInfo)
	fmt.Printf("TXID: %s\n", txInfo.TXId)
	fmt.Printf("Confirmações: %d (0 = ainda no mempool, aguardando ser minada)\n", txInfo.Confirmations)
	fmt.Printf("Taxa paga: %f BTC\n", txInfo.Fee)
	fmt.Printf("Recebida em: %v\n", txInfo.Timereceived)

	// ========== PASSO 11: MINERAR 1 BLOCO PARA CONFIRMAR ==========
	// Analogia: Como em Regtest você controla os mineradores, você mesmo minera um bloco.
	// Isso coloca sua transação dentro de um bloco, dando-lhe 1 confirmação.
	// RPC: generatetoaddress
	// Analogia: "Ei nó, minera 1 bloco e dá a recompensa para este endereço"
	fmt.Println("\n========== PASSO 11: MINERANDO 1 BLOCO ==========")
	mineParams := []any{1, *changeAddrStr}
	mineRaw := rpcCallResult(rpcURL, *rpcUser, *rpcPass, "generatetoaddress", mineParams)
	var blockHashes []string
	mustRPC(mineRaw, &blockHashes)
	for i, hash := range blockHashes {
		fmt.Printf("Bloco %d minado: %s\n", i+1, hash)
	}
	fmt.Println("Sua transação agora tem 1 confirmação!")

	// ========== PASSO 12: VERIFICAR CONFIRMAÇÕES APÓS MINING ==========
	// Analogia: Você liga de novo para o correio e pergunta se a carta já foi entregue.
	// Desta vez ela deve estar em um bloco (ter confirmação).
	// RPC: gettransaction (novamente)
	fmt.Println("\n========== PASSO 12: VERIFICANDO CONFIRMAÇÕES FINAIS ==========")
	var txInfoFinal struct {
		TXId          string  `json:"txid"`
		Confirmations int     `json:"confirmations"`
		BlockHash     string  `json:"blockhash"`
		Fee           float64 `json:"fee"`
	}
	mustRPC(rpcCallResult(rpcURL, *rpcUser, *rpcPass, "gettransaction", []any{txidHex}), &txInfoFinal)
	fmt.Printf("TXID: %s\n", txInfoFinal.TXId)
	fmt.Printf("Confirmações: %d (agora tem pelo menos 1!)\n", txInfoFinal.Confirmations)
	if txInfoFinal.BlockHash != "" {
		fmt.Printf("Incluída no bloco: %s\n", txInfoFinal.BlockHash)
	}
	fmt.Printf("Taxa paga: %f BTC\n", txInfoFinal.Fee)
	fmt.Println("\n✓ FLUXO COMPLETO FINALIZADO COM SUCESSO!")

}

// ========== HELPER: rpcCallResult ==========
// Esta função trata a comunicação com o nó Bitcoin via RPC (Remote Procedure Call)
// Analogia: É como fazer uma ligação telefônica para o nó e pedir uma informação
func rpcCallResult(url, user, pass, method string, params any) json.RawMessage {
	// Preparar o "pacote" que vai ser enviado
	// JSON-RPC é um protocolo padronizado para chamar funções remotamente
	body := map[string]any{
		"jsonrpc": "1.0",  // versão do protocolo
		"id":      "go",   // identificador da requisição
		"method":  method, // qual função remotea chamar (ex: "listunspent")
		"params":  params, // argumentos para a função
	}
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}

	// Criar requisição HTTP POST
	// (JSON-RPC roda sobre HTTP)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		panic(err)
	}

	// Adicionar autenticação
	// Username e password para acessar a API do nó
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")

	// Enviar a requisição e receber resposta
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Ler o corpo da resposta
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Decodificar a resposta JSON
	var rpcResp struct {
		Result json.RawMessage `json:"result"` // o resultado da função remota
		Error  any             `json:"error"`  // se houve erro, aqui aparece
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		panic(err)
	}

	// Verificar se o nó retornou um erro
	if rpcResp.Error != nil {
		panic(fmt.Sprintf("rpc error: %v", rpcResp.Error))
	}

	// Retornar o resultado em formato JSON bruto
	// (o chamador vai converter para a estrutura que espera)
	return rpcResp.Result
}

// ========== HELPER: mustRPC ==========
// Esta função converte o resultado JSON bruto para a estrutura esperada
// Analogia: É como traduzir a resposta do nó para uma linguagem que você entende
func mustRPC(raw json.RawMessage, target any) {
	// Decodificar o JSON em uma estrutura Go
	if err := json.Unmarshal(raw, target); err != nil {
		panic(err)
	}
}
