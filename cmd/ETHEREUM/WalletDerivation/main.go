package main

import (
	"encoding/hex"
	"fmt"
	"log"

	bip39 "github.com/tyler-smith/go-bip39"

	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func main() {
	// ========== PASSO 0: ENTENDER O PROBLEMA ==========
	// Analogia: Você quer criar uma carteira Ethereum que possa gerar múltiplos endereços
	// de forma determinística (sempre os mesmos endereços a partir da mesma "senha").
	// Isso é o que faz uma carteira HD (Hierarchical Deterministic).
	//
	// Diferença Bitcoin vs Ethereum:
	//   - Bitcoin: usa BIP32 nativo para derivação
	//   - Ethereum: usa BIP32 (igual Bitcoin) MAS converte a chave para secp256k1 e depois para Ethereum
	//
	// Sem HD: você geraria chaves aleatórias, com risco de perder a carteira se perder a seed.
	// Com HD: você gera uma única seed e dela deriva quantos endereços quiser.

	// ========== PASSO 1: INPUT - A FRASE MNEMÔNICA (SEED PHRASE) ==========
	// Analogia: É como uma frase secreta em português/inglês que você memoriza.
	// Ela representa números grandes em formato humanamente legível.
	//
	// Exemplo aqui: as 12 palavras "abandon abandon abandon..."
	// são a carteira de teste padrão (todos conhecem, usar só para estudos!)
	//
	// Estrutura BIP39:
	//   - 12 palavras = 128 bits de entropia (segurança mínima)
	//   - 24 palavras = 256 bits de entropia (máxima segurança)

	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	passphrase := "" // Opcional: camada extra de segurança (deixar vazio = sem passphrase)

	fmt.Println("========== PASSO 1: MNEMÔNICO ==========")
	fmt.Printf("Frase mnemônica: %s\n", mnemonic)
	fmt.Printf("Passphrase: %s (vazio = sem camada extra)\n\n", passphrase)

	// ========== PASSO 2: CONVERTER MNEMÔNICO EM SEED ==========
	// Analogia: A frase mnemônica é legível, mas computacionalmente fraca.
	// Precisamos "esticar" essa frase em uma seed criptográfica forte.
	//
	// Processo (BIP39):
	//   1. Validar que a frase tem 12 ou 24 palavras
	//   2. Converter palavras em 128/256 bits
	//   3. Usar PBKDF2 para "esticar" e gerar 512 bits (64 bytes)
	//
	// Analogia: É como você escrever uma senha fraca e transformá-la em uma chave criptográfica forte.

	seed := bip39.NewSeed(mnemonic, passphrase)

	fmt.Println("========== PASSO 2: SEED (512 BITS) ==========")
	fmt.Printf("Seed (hex): %s\n", hex.EncodeToString(seed))
	fmt.Printf("Tamanho: %d bytes\n\n", len(seed))

	// ========== PASSO 3: CRIAR A CHAVE MESTRE (MASTER KEY) ==========
	// Analogia: A seed é a raiz de uma árvore. A master key é o primeiro ramo.
	// A partir dela, vamos derivar todos os outros endereços.
	//
	// Processo (BIP32):
	//   1. Usar HMAC-SHA512 para processar a seed
	//   2. Gerar uma chave privada de 256 bits (para derivar filhas)
	//   3. Gerar um chain code (para criptografia da derivação)
	//   4. Resultado: Master Extended Private Key (xprv no mainnet, tprv no testnet)

	net := &chaincfg.TestNet3Params // Usando TestNet3 (não é dinheiro real)
	master, err := hd.NewMaster(seed, net)
	if err != nil {
		log.Fatalf("erro ao criar master key: %v", err)
	}

	fmt.Println("========== PASSO 3: MASTER EXTENDED PRIVATE KEY (xprv) ==========")
	fmt.Println("Master (xprv):", master.String())
	fmt.Println("Analogia: Raiz da árvore de derivação")
	fmt.Println()

	// ========== PASSO 4: DERIVAR CAMINHO BIP44 PARA ETHEREUM (m/44'/60'/0'/0/0) ==========
	// Analogia: Imagine uma árvore genealógica. Começamos na raiz (master) e descemos pelos ramos.
	// Cada número no caminho nos leva para um "filho" diferente.
	//
	// Estrutura BIP44 (padrão para ETHEREUM):
	//   m / purpose / coin_type / account / change / address_index
	//
	// Nosso exemplo: m/44'/60'/0'/0/0
	//   - m: master (raiz)
	//   - 44': BIP44 (padrão de carteiras HD)
	//   - 60': coin type 60 = ETHEREUM (assim definido no BIP44)
	//   - 0': account 0 (primeira conta)
	//   - 0: change 0 (endereços externos, não de troco)
	//   - 0: address index 0 (primeiro endereço desta combinação)
	//
	// "Hardened" (com '): impossível derivar chaves públicas sem a privada. Mais seguro!

	fmt.Println("========== PASSO 4: DERIVAÇÃO BIP44 ETHEREUM (m/44'/60'/0'/0/0) ==========")
	fmt.Println("Caminho: m / 44' / 60' / 0' / 0 / 0")
	fmt.Println("Significado:")
	fmt.Println("  - 44' = BIP44 (padrão HD wallet)")
	fmt.Println("  - 60' = Ethereum (coin type segundo BIP44)")
	fmt.Println("  - 0'  = Primeira conta")
	fmt.Println("  - 0   = Endereço externo (não troco)")
	fmt.Println("  - 0   = Primeiro endereço")
	fmt.Println()

	// Derivar cada nível do caminho
	// Nível 1: purpose (44')
	purpose, err := master.Derive(hd.HardenedKeyStart + 44)
	if err != nil {
		log.Fatalf("erro ao derivar purpose: %v", err)
	}
	fmt.Println("✓ Derivado: m/44'")

	// Nível 2: coin_type (60' = ethereum)
	coinType, err := purpose.Derive(hd.HardenedKeyStart + 60)
	if err != nil {
		log.Fatalf("erro ao derivar coin type: %v", err)
	}
	fmt.Println("✓ Derivado: m/44'/60'")

	// Nível 3: account (0' = primeira conta)
	account, err := coinType.Derive(hd.HardenedKeyStart + 0)
	if err != nil {
		log.Fatalf("erro ao derivar account: %v", err)
	}
	fmt.Println("✓ Derivado: m/44'/60'/0'")

	// Nível 4: change (0 = externo, não-hardened)
	change, err := account.Derive(0)
	if err != nil {
		log.Fatalf("erro ao derivar change: %v", err)
	}
	fmt.Println("✓ Derivado: m/44'/60'/0'/0")

	// Nível 5: address index (0 = primeiro endereço)
	addressIndex, err := change.Derive(0)
	if err != nil {
		log.Fatalf("erro ao derivar address index: %v", err)
	}
	fmt.Println("✓ Derivado: m/44'/60'/0'/0/0")
	fmt.Println()

	// ========== PASSO 5: EXTRAIR CHAVE PRIVADA ==========
	// Analogia: Agora temos a chave final no caminho. Dele extraímos a chave privada.
	//
	// Diferença de Ethereum:
	//   - Bitcoin: usa a chave privada do BIP32 diretamente (32 bytes)
	//   - Ethereum: também usa os 32 bytes de chave privada com secp256k1

	xprv := addressIndex
	privKeyECDSA, err := xprv.ECPrivKey()
	if err != nil {
		log.Fatalf("erro ao extrair chave privada: %v", err)
	}

	privKeyBytes := privKeyECDSA.Serialize()

	fmt.Println("========== PASSO 5: CHAVE PRIVADA (32 BYTES) ==========")
	privKeyHex := hex.EncodeToString(privKeyBytes)
	fmt.Println("Chave privada (hex):", privKeyHex)
	fmt.Println("  → Nunca compartilhe! Quem tiver isso controla seus ETH")
	fmt.Println()

	// ========== PASSO 6: DERIVAR CHAVE PÚBLICA ETHEREUM ==========
	// Analogia: A chave privada é transformada em uma chave pública através de secp256k1.
	// A chave pública é então hasheada para gerar um endereço Ethereum (20 bytes).
	//
	// Processo Ethereum (diferente do Bitcoin):
	//   1. Extrair chave pública secp256k1 da chave privada (65 bytes descomprimida)
	//   2. Fazer Keccak256 da chave pública descomprimida
	//   3. Pegar os últimos 20 bytes e prefixar com 0x
	//   4. Resultado: endereço Ethereum com checksum (EIP-55)
	//
	// NOTA: Para derivar o endereço Ethereum completo, seria necessário a biblioteca
	// go-ethereum e uma implementação de Keccak256. Este exemplo mostra a estrutura.

	fmt.Println("========== PASSO 6: ENDEREÇO ETHEREUM ==========")

	// Extrair chave pública serializando sem compressão (formato Ethereum)
	pubKeyUncompressed := privKeyECDSA.PubKey().SerializeUncompressed()

	fmt.Printf("✓ Chave pública não-comprimida (65 bytes): %s\n", hex.EncodeToString(pubKeyUncompressed))
	fmt.Println()

	fmt.Println("NOTA: Para completar a derivação do endereço Ethereum:")
	fmt.Println("  1. Aplicar Keccak256 na chave pública (sem prefixo 04)")
	fmt.Println("  2. Pegar os últimos 20 bytes")
	fmt.Println("  3. Aplicar checksum EIP-55")
	fmt.Println()

	fmt.Println("========== RESULTADO INTERMEDIÁRIO ==========")
	fmt.Println("Chave privada:", privKeyHex)
	fmt.Println("Chave pública (hex):", hex.EncodeToString(pubKeyUncompressed))
	fmt.Println()

	// ========== RESUMO DO FLUXO COMPLETO ==========
	fmt.Println("========== RESUMO DO FLUXO COMPLETO ==========")
	fmt.Println("1. Frase mnemônica (12 palavras)")
	fmt.Println("   ↓")
	fmt.Println("2. Seed (512 bits) via BIP39")
	fmt.Println("   ↓")
	fmt.Println("3. Master key (xprv) via BIP32")
	fmt.Println("   ↓")
	fmt.Println("4. Derivação BIP44 → m/44'/60'/0'/0/0")
	fmt.Println("   ↓")
	fmt.Println("5. Chave privada (32 bytes)")
	fmt.Println("   ↓")
	fmt.Println("6. Chave pública secp256k1 (65 bytes descomprimida)")
	fmt.Println("   ↓")
	fmt.Println("7. Keccak256(chave pública) → Endereço Ethereum (20 bytes)")
	fmt.Println("   ↓")
	fmt.Println("8. PRONTO PARA RECEBER ETH!")
	fmt.Println()
	fmt.Println("Diferença Bitcoin vs Ethereum:")
	fmt.Println("  - Bitcoin: BIP32 → BIP84 → bech32 (33 bytes comprimido → 20 bytes hash → endereço)")
	fmt.Println("  - Ethereum: BIP39/BIP44 → secp256k1 → Keccak256 → EIP-55 (65 bytes → Keccak → 20 bytes → checksum)")
}
