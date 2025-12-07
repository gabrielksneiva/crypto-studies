package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strings"

	bip39 "github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ed25519"

	"github.com/mr-tron/base58"
)

// Implementação prática de SLIP-0010 para ed25519.
// Usamos a seed BIP39 como entrada e derivamos a chave mestre com HMAC-SHA512
// usando a key "ed25519 seed". Em seguida aplicamos derivações hardened
// (necessárias para ed25519) para caminhar pelo path BIP44 padrão para Solana:
// m/44'/501'/account'/change'/index'

func slip10MasterKey(seed []byte) (k, c []byte) {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	return I[:32], I[32:]
}

func slip10ChildKey(parentK, parentC []byte, index uint32) (k, c []byte) {
	// Only hardened
	buf := make([]byte, 1+32+4)
	buf[0] = 0x00
	copy(buf[1:33], parentK)
	binary.BigEndian.PutUint32(buf[33:], index)
	mac := hmac.New(sha512.New, parentC)
	mac.Write(buf)
	I := mac.Sum(nil)
	return I[:32], I[32:]
}

func parsePath(path string) ([]uint32, error) {
	// Expect formats like: m/44'/501'/0'/0'/0'
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] != "m" {
		return nil, fmt.Errorf("path deve iniciar com 'm'")
	}
	var res []uint32
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		hardened := strings.HasSuffix(p, "'") || strings.HasSuffix(p, "h")
		numStr := strings.TrimRight(strings.TrimRight(p, "'"), "h")
		var n uint32
		_, err := fmt.Sscanf(numStr, "%d", &n)
		if err != nil {
			return nil, err
		}
		if hardened {
			n = n | 0x80000000
		}
		res = append(res, n)
	}
	return res, nil
}

func main() {
	// Flags para permitir entrada via CLI
	mnemonic := flag.String("mnemonic", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "frase mnemônica (BIP39)")
	passphrase := flag.String("passphrase", "", "passphrase BIP39 (opcional)")
	path := flag.String("path", "m/44'/501'/0'/0'/0'", "caminho de derivação (ex: m/44'/501'/0'/0'/0')")
	flag.Parse()

	fmt.Println("========== WalletDerivation (Solana) ==========")
	fmt.Println("Usando BIP39 + SLIP-0010 (ed25519) para derivação prática")

	// Gerar seed a partir do mnemônico
	seed := bip39.NewSeed(*mnemonic, *passphrase)
	fmt.Printf("Seed BIP39 (hex): %s\n", hex.EncodeToString(seed))

	// Master key
	k, c := slip10MasterKey(seed)

	// Parse path e derivar iterativamente
	idxs, err := parsePath(*path)
	if err != nil {
		log.Fatalf("erro ao parsear path: %v", err)
	}
	for i, ix := range idxs {
		k, c = slip10ChildKey(k, c, ix)
		fmt.Printf("Derivado nível %d (index: %d)\n", i+1, ix&^0x80000000)
	}

	// k agora é a seed de 32 bytes compatível com ed25519
	if len(k) != 32 {
		log.Fatalf("tamanho de chave derivada inesperado: %d", len(k))
	}

	priv := ed25519.NewKeyFromSeed(k)
	pub := priv.Public().(ed25519.PublicKey)

	fmt.Println("========== Resultado ==========")
	fmt.Printf("Chave privada (seed 32 bytes hex): %s\n", hex.EncodeToString(k))
	fmt.Printf("Chave privada (ed25519, hex): %s\n", hex.EncodeToString(priv))
	fmt.Printf("Chave pública (ed25519, hex): %s\n", hex.EncodeToString(pub))
	fmt.Printf("Endereço (base58): %s\n", base58.Encode(pub))
	fmt.Println("\nObservação: este fluxo implementa SLIP-0010 corretamente para ed25519.")
}
