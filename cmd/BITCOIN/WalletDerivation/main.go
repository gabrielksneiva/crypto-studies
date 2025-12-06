package main

import (
	"encoding/hex"
	"fmt"
	"log"

	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/btcsuite/btcd/btcutil"
	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func main() {
	// ========== PASSO 0: ENTENDER O PROBLEMA ==========
	// Analogia: Você quer criar uma carteira Bitcoin que possa gerar múltiplos endereços
	// de forma determinística (sempre os mesmos endereços a partir da mesma "senha").
	// Isso é o que faz uma carteira HD (Hierarchical Deterministic).
	//
	// Sem HD: você geraria chaves aleatórias, com risco de perder a carteira se perder a seed.
	// Com HD: você gera uma única seed e dela deriva quantos endereços quiser.

	// ========== PASSO 1: INPUT - A FRASE MNEMÔNICA (SEED PHRASE) ==========
	// Analogia: É como uma frase secreta em português/inglês que você memoriza.
	// Ela representa números grandes em formato humanamente legível.
	//
	// Exemplo aqui: as 12 palavras "abandon abandon abandon..."
	// são a carteira de teste padrão do Bitcoin (todos conhecem, usar só para estudos!)
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

	// ========== PASSO 4: DERIVAR CAMINHO BIP84 (m/84'/1'/0'/0/0) ==========
	// Analogia: Imagine uma árvore genealógica. Começamos na raiz (master) e descemos pelos ramos.
	// Cada número no caminho nos leva para um "filho" diferente.
	//
	// Estrutura BIP44/BIP84 (padrão de carteiras modernas):
	//   m / purpose / coin_type / account / change / address_index
	//
	// Nosso exemplo: m/84'/1'/0'/0/0
	//   - m: master (raiz)
	//   - 84': BIP84 (endereços SegWit v0 = bech32) — ' significa "hardened"
	//   - 1': coin type 1 = Bitcoin Testnet
	//   - 0': account 0 (primeira conta)
	//   - 0: change 0 (endereços externos, não de troco)
	//   - 0: address index 0 (primeiro endereço desta combinação)
	//
	// "Hardened" (com '): impossível derivar chaves públicas sem a privada. Mais seguro!

	fmt.Println("========== PASSO 4: DERIVAÇÃO BIP84 (m/84'/1'/0'/0/0) ==========")
	fmt.Println("Caminho: m / 84' / 1' / 0' / 0 / 0")
	fmt.Println("Significado:")
	fmt.Println("  - 84' = SegWit v0 (bech32)")
	fmt.Println("  - 1'  = Bitcoin Testnet")
	fmt.Println("  - 0'  = Primeira conta")
	fmt.Println("  - 0   = Endereço externo (não troco)")
	fmt.Println("  - 0   = Primeiro endereço")
	fmt.Println()

	// Derivar cada nível do caminho
	// Nível 1: purpose (84')
	purpose, err := master.Derive(hd.HardenedKeyStart + 84)
	if err != nil {
		log.Fatalf("erro ao derivar purpose: %v", err)
	}
	fmt.Println("✓ Derivado: m/84'")

	// Nível 2: coin_type (1' = testnet)
	coinType, err := purpose.Derive(hd.HardenedKeyStart + 1)
	if err != nil {
		log.Fatalf("erro ao derivar coin type: %v", err)
	}
	fmt.Println("✓ Derivado: m/84'/1'")

	// Nível 3: account (0' = primeira conta)
	account, err := coinType.Derive(hd.HardenedKeyStart + 0)
	if err != nil {
		log.Fatalf("erro ao derivar account: %v", err)
	}
	fmt.Println("✓ Derivado: m/84'/1'/0'")

	// Nível 4: change (0 = externo, não-hardened)
	change, err := account.Derive(0)
	if err != nil {
		log.Fatalf("erro ao derivar change: %v", err)
	}
	fmt.Println("✓ Derivado: m/84'/1'/0'/0")

	// Nível 5: address index (0 = primeiro endereço)
	addressIndex, err := change.Derive(0)
	if err != nil {
		log.Fatalf("erro ao derivar address index: %v", err)
	}
	fmt.Println("✓ Derivado: m/84'/1'/0'/0/0")
	fmt.Println()

	// ========== PASSO 5: EXTRAIR CHAVE PRIVADA E PÚBLICA ==========
	// Analogia: Agora temos a chave final no caminho. Dele extraímos:
	//   - Chave privada (xprv) = seu segredo, nunca compartilhar!
	//   - Chave pública (xpub) = pode ser compartilhada (ex: para pedir doações)
	//
	// Operação Neuter: "Neutralizar" remove a chave privada, mantendo só a pública
	// É como tirar a combinação de um cofre mas manter a capacidade de colocar coisas nele.

	xprv := addressIndex
	xpub, err := xprv.Neuter()
	if err != nil {
		log.Fatalf("erro em Neuter(): %v", err)
	}

	fmt.Println("========== PASSO 5: CHAVES PRIVADA E PÚBLICA ==========")
	fmt.Println("Derived xprv (privada):", xprv.String())
	fmt.Println("  → Nunca compartilhe! Quem tiver isso controla seus bitcoins")
	fmt.Println("\nDerived xpub (pública):", xpub.String())
	fmt.Println("  → Pode compartilhar. Permite apenas receber, não gastar")
	fmt.Println()

	// ========== PASSO 6: GERAR ENDEREÇO P2WPKH BECH32 ==========
	// Analogia: A chave pública é como o número de conta do banco.
	// O endereço é um "IBAN" (código mais curto e fácil de usar).
	//
	// Processo:
	//   1. Extrair chave pública da chave estendida
	//   2. Comprimir a chave pública (33 bytes)
	//   3. Hash160 (SHA256 + RIPEMD160) → 20 bytes
	//   4. Envolver em script P2WPKH (Pay-to-Witness-PubKeyHash)
	//   5. Converter para bech32 (formato legível)
	//
	// Formato: bcrt1q... (bc = Bitcoin, rt = regtest, q = tipo de script)

	fmt.Println("========== PASSO 6: ENDEREÇO BECH32 (P2WPKH) ==========")

	// Extrair chave pública
	pubKey, err := xpub.ECPubKey()
	if err != nil {
		log.Fatalf("erro ao extrair public key: %v", err)
	}
	fmt.Println("✓ Chave pública extraída")

	// Comprimir (33 bytes)
	serialized := pubKey.SerializeCompressed()
	fmt.Printf("✓ Chave pública comprimida: %s\n", hex.EncodeToString(serialized))

	// Hash160 (20 bytes)
	pubKeyHash := btcutil.Hash160(serialized)
	fmt.Printf("✓ Hash160 da chave pública: %s\n", hex.EncodeToString(pubKeyHash))

	// Criar endereço P2WPKH
	addr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, net)
	if err != nil {
		log.Fatalf("erro ao criar endereco: %v", err)
	}

	fmt.Println("\n========== RESULTADO FINAL ==========")
	fmt.Println("Endereço P2WPKH bech32:", addr.EncodeAddress())
	fmt.Println("\n✓ ESTE É O SEU ENDEREÇO DE RECEBIMENTO!")
	fmt.Println("   Qualquer pessoa pode enviar bitcoins para este endereço.")
	fmt.Println("   Mas só você (com a chave privada) pode gastar.")
	fmt.Println()

	// ========== RESUMO DO FLUXO COMPLETO ==========
	fmt.Println("========== RESUMO DO FLUXO COMPLETO ==========")
	fmt.Println("1. Frase mnemônica (12 palavras)")
	fmt.Println("   ↓")
	fmt.Println("2. Seed (512 bits) via BIP39")
	fmt.Println("   ↓")
	fmt.Println("3. Master key (xprv) via BIP32")
	fmt.Println("   ↓")
	fmt.Println("4. Derivação BIP84 → m/84'/1'/0'/0/0")
	fmt.Println("   ↓")
	fmt.Println("5. Chaves privada (xprv) e pública (xpub)")
	fmt.Println("   ↓")
	fmt.Println("6. Endereço bech32 (P2WPKH)")
	fmt.Println("   ↓")
	fmt.Println("7. PRONTO PARA RECEBER BITCOINS!")
	fmt.Println("\nAnalogia: Assim como um banco cria sua conta a partir do seu CPF,")
	fmt.Println("o Bitcoin cria seus endereços a partir de uma frase que você memoriza.")
}
