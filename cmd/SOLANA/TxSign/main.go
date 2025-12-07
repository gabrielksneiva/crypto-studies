package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	bip39 "github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ed25519"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

// ============================================================================
// TxSign SOLANA - versão didática estilo 'aula'
// - Comentários em PT-BR, com analogias e explicações linha-a-linha.
// - Deriva chaves via BIP39 -> SLIP-0010 (ed25519), monta e assina transação,
//   e opcionalmente envia ao RPC (ex.: solana-test-validator).
// ============================================================================

// slip10MasterKey: gera (k, c) master a partir da seed BIP39
func slip10MasterKey(seed []byte) (k, c []byte) {
	// HMAC-SHA512 com key "ed25519 seed" conforme SLIP-0010
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	// k = IL (32 bytes), c = IR (32 bytes)
	return I[:32], I[32:]
}

// slip10ChildKey: deriva um filho hardened (somente hardened usado em ed25519)
func slip10ChildKey(parentK, parentC []byte, index uint32) (k, c []byte) {
	// Montamos: 0x00 || parentK || index (4 bytes big-endian)
	buf := make([]byte, 1+32+4)
	buf[0] = 0x00
	copy(buf[1:33], parentK)
	binary.BigEndian.PutUint32(buf[33:], index)
	mac := hmac.New(sha512.New, parentC)
	mac.Write(buf)
	I := mac.Sum(nil)
	return I[:32], I[32:]
}

// parsePath: converte string como "m/44'/501'/0'/0'/0'" em índices uint32 (com bit hardened)
func parsePath(path string) ([]uint32, error) {
	// Analogia: O path é como um mapa de níveis (raiz 'm' -> contas -> índices).
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

// main: montar, assinar e (opcional) broadcast uma transferência Solana
func main() {
	// FLAGS: entrada do programa (mnemonic, path, rpc, destino, valor)
	mnemonic := flag.String("mnemonic", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "frase mnemônica BIP39")
	passphrase := flag.String("passphrase", "", "passphrase BIP39 (opcional)")
	path := flag.String("path", "m/44'/501'/0'/0'/0'", "caminho de derivação (hardened) para a conta")
	rpcURL := flag.String("rpc", "http://127.0.0.1:8899", "RPC endpoint (ex: http://127.0.0.1:8899 para solana-test-validator)")
	dest := flag.String("to", "", "endereço de destino (base58)")
	lamports := flag.Uint64("lamports", 1000000000, "quantidade em lamports (1 SOL = 1_000_000_000 lamports)")
	broadcast := flag.Bool("broadcast", false, "se true, envia a transação ao RPC; caso contrário só imprime RAW")
	flag.Parse()

	// Validação simples
	if *dest == "" {
		log.Fatal("forneça -to <destino> (endereço base58)")
	}

	// Aula: primeiro passo sempre é derivar a chave da seed (BIP39 → SLIP-0010)
	fmt.Println("========== SOLANA TxSign - Aula Didática ==========")
	fmt.Println("Passo 1: derivação da chave via BIP39 + SLIP-0010 (ed25519)")

	// Gerar seed a partir das palavras (BIP39)
	seed := bip39.NewSeed(*mnemonic, *passphrase)
	fmt.Printf("Seed BIP39 (hex): %s\n", hex.EncodeToString(seed))

	// Master key
	k, c := slip10MasterKey(seed)

	// Parse do caminho e derivação iterativa (cada nível é hardened para ed25519)
	idxs, err := parsePath(*path)
	if err != nil {
		log.Fatalf("erro ao parsear path: %v", err)
	}
	for i, ix := range idxs {
		k, c = slip10ChildKey(k, c, ix)
		// Didática: informamos cada nível derivado
		fmt.Printf("Derivado nível %d -> index (raw): %d\n", i+1, ix&^0x80000000)
	}

	// Agora k é a seed de 32 bytes compatível com ed25519
	if len(k) != 32 {
		log.Fatalf("chave derivada com tamanho inesperado: %d", len(k))
	}

	// Gerar par chave privada/pública Ed25519
	priv := ed25519.NewKeyFromSeed(k)
	pub := priv.Public().(ed25519.PublicKey)

	// Transformar em PublicKey do pacote solana-go
	from := solana.PublicKeyFromBytes(pub)
	to, err := solana.PublicKeyFromBase58(*dest)
	if err != nil {
		log.Fatalf("dest inválido: %v", err)
	}

	fmt.Println("\nPasso 2: montar a instrução de transferência (system program)")
	// Cria instrução simples de transferência
	// Usamos o helper específico do pacote `programs/system`.
	// A função retorna um builder que precisa ser "buildado" em um Instruction.
	instr := system.NewTransferInstruction(*lamports, from, to).Build()

	// Conectar ao RPC local (solana-test-validator por padrão)
	client := rpc.New(*rpcURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Obter blockhash mais recente (necessário para construir a transação)
	bh, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("erro ao obter blockhash: %v", err)
	}

	// Montar a transação usando o helper NewTransaction:
	// - primeiro argumento: slice de instruções
	// - segundo: recent blockhash
	tx, err := solana.NewTransaction([]solana.Instruction{instr}, bh.Value.Blockhash)
	if err != nil {
		log.Fatalf("erro ao montar transacao: %v", err)
	}

	// Assinar a transação localmente com a chave privada (ed25519)
	// A API pede um getter que, dado um PublicKey, retorna *solana.PrivateKey correspondente.
	getter := func(key solana.PublicKey) *solana.PrivateKey {
		// Converte nossa chave ed25519.PrivateKey (64 bytes) para solana.PrivateKey
		pk := solana.PrivateKey(priv)
		// Se a chave pública corresponder ao solicitado, retornamos o ponteiro
		if pk.PublicKey().Equals(key) {
			return &pk
		}
		return nil
	}

	// Signatures: tx.Sign exige o getter e retorna as assinaturas aplicadas.
	_, err = tx.Sign(getter)
	if err != nil {
		log.Fatalf("erro ao assinar transacao: %v", err)
	}

	// Serializar em bytes (RAW) usando MarshalBinary
	raw, err := tx.MarshalBinary()
	if err != nil {
		log.Fatalf("erro ao serializar tx: %v", err)
	}

	fmt.Println("\n=== RAW TX (hex) ===")
	fmt.Println(hex.EncodeToString(raw))

	// Broadcast opcional: enviar ao RPC local
	if *broadcast {
		sig, err := client.SendRawTransaction(ctx, raw)
		if err != nil {
			log.Fatalf("erro ao enviar transacao: %v", err)
		}
		fmt.Printf("Transação enviada. Signature: %s\n", sig)
	} else {
		fmt.Println("Broadcast não solicitado. Rode com -broadcast para enviar a transação ao RPC local.")
	}
}
