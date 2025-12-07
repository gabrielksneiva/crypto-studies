# CryptoStudys - Estudos Práticos de Bitcoin, Ethereum e Solana

Repositório educacional para aprender na prática como funcionam transações em Bitcoin, Ethereum e Solana, desde a derivação de carteiras HD até a assinatura e broadcast de transações.

## 📚 Estrutura do Projeto

```
cmd/
├── BITCOIN/
│   ├── WalletDerivation/    # Derivação de carteira Bitcoin (BIP39/BIP32/BIP84)
│   └── TxSign/              # Criação, assinatura e broadcast de transações Bitcoin
└── ETHEREUM/
    ├── WalletDerivation/    # Derivação de carteira Ethereum (BIP39/BIP32/BIP44)
    └── TxSign/              # Criação, assinatura e broadcast de transações Ethereum
└── SOLANA/
   ├── WalletDerivation/    # Derivação de carteira Solana (BIP39 -> SLIP-0010 -> ed25519)
   └── TxSign/              # Montagem, assinatura e broadcast de transações Solana (system transfer)
```

## 🎯 O que você vai aprender

### Bitcoin
- ✅ Como derivar endereços Bitcoin usando BIP39/BIP32/BIP84
- ✅ Diferença entre hardened e non-hardened derivation
- ✅ Como criar uma transação Bitcoin manualmente
- ✅ Como assinar transações usando RPC do bitcoind
- ✅ UTXO model (modelo de cédulas)
- ✅ Auto-descoberta de UTXOs e endereços de troco
- ✅ Broadcast e verificação de confirmações

### Ethereum
- ✅ Como derivar endereços Ethereum usando BIP39/BIP32/BIP44
- ✅ Diferença entre Bitcoin (coin type 1) e Ethereum (coin type 60)
- ✅ Como criar uma transação Ethereum manualmente
- ✅ Como assinar transações com ECDSA (v, r, s)
- ✅ Account model vs UTXO model
- ✅ Auto-descoberta de nonce e gas price
- ✅ Broadcast e verificação de confirmações na rede
- ✅ EIP-1559 (base fee + priority fee) e estimativa automática de gas

### Solana
- ✅ Como derivar endereços Solana usando BIP39 -> SLIP-0010 (ed25519)
- ✅ Porque usamos derivação hardened para ed25519
- ✅ Formato de endereços (base58) e relação com `ed25519.PublicKey`
- ✅ Como montar uma instrução `system.Transfer` e criar/assinar a transação
- ✅ Testar localmente com `solana-test-validator` e usar `solana airdrop` para prover fundos

## 🚀 Pré-requisitos

### Para Bitcoin
1. **Go 1.23+** instalado
2. **Bitcoin Core** (bitcoind) rodando em modo regtest:
   ```bash
   # Linux/Mac
   bitcoind -regtest -daemon -rpcuser=admin -rpcpassword=123456
   
   # Criar carteira
   bitcoin-cli -regtest createwallet "wallet"
   
   # Gerar alguns blocos para ter fundos
   bitcoin-cli -regtest -generate 101
   ```

### Para Ethereum
1. **Go 1.23+** instalado
2. **Anvil** (Foundry) ou outro nó Ethereum local:
   ```bash
   # Instalar Foundry (inclui Anvil)
   curl -L https://foundry.paradigm.xyz | bash
   foundryup
   
   # Rodar Anvil (cria rede Ethereum local)
   anvil
   ```

## 📖 Guias de Uso

### 1. Bitcoin - Derivação de Carteira

```bash
# Compilar
go build -o btc_wallet ./cmd/BITCOIN/WalletDerivation

# Executar
./btc_wallet
```

**Saída esperada:**
- Frase mnemônica (12 palavras)
- Seed (512 bits)
- Master key (xprv)
- Derivação completa: m/84'/1'/0'/0/0
- Endereço P2WPKH bech32 (bcrt1q...)

### 2. Bitcoin - Assinatura de Transação

```bash
# Compilar
go build -o btc_sign ./cmd/BITCOIN/TxSign

# Executar com auto-descoberta (só precisa informar destino e valor)
./btc_sign -dest bcrt1qafjfjrfxnkd9gwtcuahwl7v4ucf6cjg0hs6cz7 -send_sat 100000000

# Flags disponíveis:
#   -rpcuser      : usuário RPC do bitcoind (padrão: admin)
#   -rpcpass      : senha RPC do bitcoind (padrão: 123456)
#   -rpchost      : host:port do bitcoind (padrão: 127.0.0.1:18443)
#   -wallet       : nome da carteira (padrão: wallet)
#   -dest         : endereço de destino (obrigatório)
#   -send_sat     : quantidade em satoshis (padrão: 100000000 = 1 BTC)
#   -fee_sat      : taxa em satoshis (padrão: 500)
```

**O programa faz automaticamente:**
1. ✅ Busca UTXOs disponíveis na carteira
2. ✅ Seleciona o primeiro UTXO com saldo suficiente
3. ✅ Gera endereço de troco automaticamente
4. ✅ Monta a transação
5. ✅ Assina via RPC do bitcoind
6. ✅ Faz broadcast para a rede
7. ✅ Verifica se está na mempool
8. ✅ Minera um bloco automaticamente
9. ✅ Verifica confirmação

### 3. Ethereum - Derivação de Carteira

```bash
# Compilar
go build -o eth_wallet ./cmd/ETHEREUM/WalletDerivation

# Executar
./eth_wallet
```

**Saída esperada:**
- Frase mnemônica (12 palavras)
- Seed (512 bits)
- Master key (xprv)
- Derivação completa: m/44'/60'/0'/0/0
- Chave privada (32 bytes)
- Chave pública não-comprimida (65 bytes)
- Nota sobre Keccak256 para endereço final

### 4. Ethereum - Assinatura de Transação

```bash
# Compilar
go build -o eth_sign ./cmd/ETHEREUM/TxSign

# Executar com auto-descoberta
./eth_sign -to 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 -value 0.5

# Flags disponíveis:
#   -rpc          : RPC endpoint (padrão: http://127.0.0.1:8545)
#   -chain        : Chain ID (padrão: 31337 = anvil local)
#   -mnemonic     : frase BIP39 (padrão: test test test...)
#   -account      : índice da conta (padrão: 0)
#   -to           : endereço de destino (obrigatório)
#   -value        : quantidade de ETH (padrão: 0.1)
#   -gas          : gas limit (padrão: 21000)
#   -gasprice     : gas price em wei (0 = auto, padrão: 0)
#   -nonce        : nonce (-1 = auto, padrão: -1)
```

**O programa faz automaticamente:**
1. ✅ Deriva chave privada do mnemônico
2. ✅ Conecta no nó Ethereum
3. ✅ Busca nonce automaticamente (se não informado)
4. ✅ Busca gas price automaticamente (se não informado)
5. ✅ Monta a transação
6. ✅ Assina com ECDSA (v, r, s)
7. ✅ Faz broadcast para a rede
8. ✅ Verifica na mempool
9. ✅ Aguarda confirmação (até 60 segundos)
10. ✅ Mostra resultado (sucesso/falha)

### 5. Solana - Derivação de Carteira

```bash
# Compilar
go build -o sol_wallet ./cmd/SOLANA/WalletDerivation

# Executar (exemplo)
./sol_wallet -mnemonic "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about" -passphrase "" -path "m/44'/501'/0'/0'/0'"
```

**Saída esperada:**
- Frase mnemônica (12 palavras) — ou a frase que você passou
- Seed BIP39 (hex)
- Master key (IL/IR da SLIP-0010)
- Derivação completa: m/44'/501'/0'/0'/0'
- Chave privada (seed 32 bytes -> ed25519.PrivateKey)
- Chave pública / endereço (base58)

Observações:
- Solana usa chaves ed25519; para derivação correta usamos SLIP-0010 (HMAC-SHA512 com key "ed25519 seed").
- O caminho padrão aqui segue BIP44 com `coin_type = 501` (Solana) e utiliza derivação hardened em níveis para compatibilidade.

### 6. Solana - Assinatura de Transação

```bash
# Compilar
go build -o sol_sign ./cmd/SOLANA/TxSign

# Executar com destino e broadcast (assumindo validador local)
./sol_sign -mnemonic "<SUA_MNEMONIC>" -to <DEST_BASE58> -lamports 1000000 -broadcast -rpc http://127.0.0.1:8899
```

**Flags disponíveis:**
- `-rpc` : RPC endpoint (padrão: `http://127.0.0.1:8899`)
- `-mnemonic` : frase BIP39 (padrão: frase de exemplo)
- `-passphrase` : passphrase BIP39 (opcional)
- `-path` : caminho de derivação (padrão: `m/44'/501'/0'/0'/0'`)
- `-to` : endereço de destino (base58) — obrigatório
- `-lamports` : quantidade em lamports (ex: `1000000` = 0.001 SOL)
- `-broadcast` : envia a transação ao RPC quando presente

**O programa faz automaticamente:**
1. ✅ Deriva chave privada do mnemônico via BIP39 -> SLIP-0010
2. ✅ Constrói instrução `system.Transfer` com `programs/system`
3. ✅ Obtém `recent blockhash` do RPC
4. ✅ Monta a transação e aplica assinatura local (ed25519)
5. ✅ Serializa a transação (raw) e imprime em hex/base64
6. ✅ Faz broadcast para o RPC se `-broadcast` for passado

Observações:
- 1 SOL = 1_000_000_000 lamports. Use `solana airdrop` no validador local para prover fundos.
- Para testes locais use `solana-test-validator` no RPC padrão `http://127.0.0.1:8899`.


## 🔧 Setup Rápido - Anvil (Ethereum)

```bash
# Terminal 1: Rodar Anvil
anvil

# Terminal 2: Enviar transação
cd /caminho/para/BTCStudys
go build -o eth_sign ./cmd/ETHEREUM/TxSign

# A conta padrão do Anvil já tem 10000 ETH
# Endereço: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

# Enviar 1 ETH para conta #1 do Anvil
./eth_sign \
  -to 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 \
  -value 1.0

# Verificar no Anvil que a transação foi minerada!
```

## 🔧 Setup Rápido - Bitcoin Regtest

```bash
# Terminal 1: Rodar bitcoind
bitcoind -regtest -daemon -rpcuser=admin -rpcpassword=123456
bitcoin-cli -regtest createwallet "wallet"
bitcoin-cli -regtest -generate 101

# Terminal 2: Enviar transação
cd /caminho/para/BTCStudys
go build -o btc_sign ./cmd/BITCOIN/TxSign

# Gerar um endereço de destino
DEST=$(bitcoin-cli -regtest getnewaddress)

# Enviar 1 BTC
./btc_sign -dest $DEST -send_sat 100000000

# Verificar transação
bitcoin-cli -regtest listtransactions
```

## 📊 Diferenças Bitcoin vs Ethereum

| Aspecto | Bitcoin | Ethereum |
|---------|---------|----------|
| **Modelo de conta** | UTXO (cédulas) | Account (saldo) |
| **Derivação** | BIP84 (m/84'/1'/0'/0/0) | BIP44 (m/44'/60'/0'/0/0) |
| **Coin type** | 1 (testnet) ou 0 (mainnet) | 60 |
| **Endereço** | bech32 (bc1q...) | Hex com checksum (0x...) |
| **Assinatura** | Via RPC do bitcoind | ECDSA direto (v, r, s) |
| **Taxa** | Satoshis por byte | Gas × Gas Price |
| **Confirmação** | ~10 minutos | ~12 segundos |
| **Linguagem** | Script (limitado) | EVM (Turing-complete) |

## 🎓 Conceitos Ensinados

### BIP39 (Mnemonic)
Converte 12 ou 24 palavras em uma seed de 512 bits. É como sua "senha mestre".

### BIP32 (HD Wallets)
Permite derivar múltiplas chaves de uma única seed. É uma árvore hierárquica.

### BIP44 (Multi-Coin)
Padrão para derivação de múltiplas criptomoedas da mesma seed.
- Bitcoin: m/44'/0'/0'/0/0 (mainnet) ou m/44'/1'/0'/0/0 (testnet)
- Ethereum: m/44'/60'/0'/0/0

### BIP84 (Native SegWit)
Padrão para endereços SegWit nativos (bech32) no Bitcoin.
- Caminho: m/84'/0'/0'/0/0 (mainnet) ou m/84'/1'/0'/0/0 (testnet)

### Hardened Derivation (')
Derivação "endurecida" que torna impossível derivar chaves filhas sem a chave privada pai.

### UTXO (Unspent Transaction Output)
Modelo de "cédulas" do Bitcoin. Cada UTXO é como uma nota de dinheiro que você pode gastar.

### Account Model
Modelo de "saldo em conta" do Ethereum. Cada endereço tem um saldo total.

### Nonce
Número sequencial para prevenir replay attacks no Ethereum.

### Gas
"Combustível" do Ethereum. Você paga gas para executar transações/contratos.

### RLP (Recursive Length Prefix)
Formato de serialização usado pelo Ethereum para codificar transações.

### Keccak256
Função hash usada pelo Ethereum (diferente do SHA256 do Bitcoin).

## 🐛 Troubleshooting

### Bitcoin: "error connecting to RPC"
```bash
# Certifique-se de que o bitcoind está rodando
bitcoin-cli -regtest getblockchaininfo

# Se não estiver rodando:
bitcoind -regtest -daemon -rpcuser=admin -rpcpassword=123456
```

### Bitcoin: "insufficient funds"
```bash
# Gere mais blocos para ter fundos
bitcoin-cli -regtest -generate 101
```

### Ethereum: "connection refused"
```bash
# Certifique-se de que Anvil está rodando
anvil

# Ou ajuste o RPC endpoint:
./eth_sign -rpc http://localhost:8545 -to 0x... -value 1.0
```

### Ethereum: "insufficient funds for gas * price + value"
```bash
# A conta padrão do mnemônico "test test test..." precisa ter ETH
# Use Anvil com o mnemônico padrão, ou
# Envie ETH para o endereço derivado primeiro
```

## 📝 Próximos Passos

## 🪙 Solana

Seção dedicada a aprender, na prática, a derivar carteiras e assinar/transmitir transações Solana.

### O que você vai aprender (Solana)
- ✅ Como derivar chaves usando BIP39 -> SLIP-0010 (ed25519)
- ✅ Porque usamos derivação hardened para ed25519
- ✅ Formato de endereços (base58) e relação com `ed25519.PublicKey`
- ✅ Como montar uma instrução `system.Transfer` e criar/assinar a transação
- ✅ Como enviar (broadcast) uma transação para um `solana-test-validator` local

### Pré-requisitos (Solana)
1. **Go 1.23+** instalado
2. **Solana Tool Suite** (opcional, mas recomendado para testes locais) — inclui `solana-test-validator` e `solana` CLI
    ```bash
    # instalar solana tool (exemplo, veja docs oficiais se necessário)
    sh -c "$(curl -sSfL https://release.solana.com/stable/install)"
    export PATH="$HOME/.local/share/solana/install/active_release/bin:$PATH"
    ```

3. (Opcional) ter `solana-test-validator` rodando localmente para testes:
    ```bash
    # Inicia validador local (RPC padrão em http://127.0.0.1:8899)
    solana-test-validator
    ```

### Estrutura / comandos úteis (Solana)
- Compilar derivação de carteira:
   ```bash
   go build -o sol_wallet ./cmd/SOLANA/WalletDerivation
   ./sol_wallet # Mostra a derivação e endereço (base58)
   ```

- Compilar TxSign (monta, assina e opcionalmente envia):
   ```bash
   go build -o sol_tx ./cmd/SOLANA/TxSign
   ```

### `WalletDerivation` — uso rápido
- Arquivo: `cmd/SOLANA/WalletDerivation/main.go`
- Objetivo: derivar chave privada/pública ed25519 a partir de mnemônico BIP39 usando SLIP-0010 e imprimir o endereço base58.
- Exemplo de execução (usa path padrão `m/44'/501'/0'/0'/0'`):
   ```bash
   ./sol_wallet -mnemonic "abandon abandon ... about" -passphrase "" -path "m/44'/501'/0'/0'/0'"
   ```
- Saída esperada:
   - Seed BIP39 (hex)
   - Chave privada (seed 32 bytes -> ed25519.PrivateKey)
   - Chave pública (base58 address)

### `TxSign` — montar/assinar/enviar
- Arquivo: `cmd/SOLANA/TxSign/main.go`
- Objetivo: a partir do mnemônico BIP39 derivar chave (SLIP-0010), criar instrução `system.Transfer`, assinar localmente e opcionalmente transmitir para um RPC.
- Flags principais:
   - `-mnemonic` : frase BIP39 (padrão: frase de exemplo)
   - `-passphrase` : passphrase BIP39 (opcional)
   - `-path` : caminho de derivação (padrão: `m/44'/501'/0'/0'/0'`)
   - `-rpc` : RPC endpoint (padrão: `http://127.0.0.1:8899`)
   - `-to` : endereço destino (base58) — obrigatório
   - `-lamports` : quantidade em lamports (ex: `1000000` = 0.001 SOL)
   - `-broadcast` : envia a transação ao RPC quando presente

#### Exemplo prático (com validador local)
1. Inicie o `solana-test-validator` em um terminal:
    ```bash
    solana-test-validator
    ```

2. Derive um endereço (ou use `WalletDerivation`) e faça um airdrop local:
    ```bash
    # supondo que ADDRESS seja o endereço base58 derivado
    solana airdrop 1 $ADDRESS --url http://127.0.0.1:8899
    ```

3. Envie uma transferência usando `TxSign` (substitua `DEST`):
    ```bash
    ./sol_tx -mnemonic "<SUA_MNEMONIC>" -to DEST -lamports 1000000 -broadcast -rpc http://127.0.0.1:8899
    ```

Observações:
- 1 SOL = 1_000_000_000 lamports. Ajuste `-lamports` conforme necessário.
- O caminho `m/44'/501'/0'/0'/0'` segue BIP44 com `coin_type = 501` (Solana), usando derivação hardened em todos os níveis para compatibilidade com ed25519 (SLIP-0010).

### Troubleshooting (Solana)
- Erro de conexão RPC:
   ```bash
   # Verifique se o validador está ligado e o RPC está acessível
   curl http://127.0.0.1:8899/  # deve responder
   ```
- Saldo insuficiente:
   ```bash
   # Faça airdrop no validador local
   solana airdrop 1 <SEU_ENDERECO> --url http://127.0.0.1:8899
   ```
- Erro ao derivar chaves / caminho inválido:
   - Verifique se o path está no formato `m/44'/501'/0'/0'/0'` e que números são válidos.

#### Erro ao executar o instalador oficial via `curl`

Se ao rodar o comando de instalação oficial você recebe:

```
curl: (35) error:0A000126:SSL routines::unexpected eof while reading
```

Possíveis causas e soluções:

- Problema de TLS/SSL local (OpenSSL/curl desatualizado ou incompatível): tente atualizar certificados e pacotes TLS:
   ```bash
   sudo apt update
   sudo apt install --reinstall ca-certificates openssl curl
   sudo update-ca-certificates
   ```

- Interceptação por proxy/filtragem corporativa (MITM) — verifique variáveis de ambiente `HTTP_PROXY`/`HTTPS_PROXY` e se estiver atrás de proxy configure corretamente:
   ```bash
   echo $HTTPS_PROXY
   export HTTPS_PROXY="http://proxy.example:8080"
   ```

- Forçar TLSv1.2 (teste):
   ```bash
   curl --tlsv1.2 -v -sSfL https://release.solana.com/stable/install -o /dev/null
   ```

- Teste de handshake com OpenSSL para inspecionar detalhes:
   ```bash
   openssl s_client -connect release.solana.com:443 -servername release.solana.com
   ```

- Workaround temporário (NÃO recomendado em produção): baixar script com verificação desativada apenas para testar:
   ```bash
   curl -k -sSfL https://release.solana.com/stable/install | sh
   ```

- Alternativa mais segura: baixar release/tarball oficial diretamente dos releases do GitHub e instalar manualmente (ver página de releases do Solana):
   ```bash
   # exemplo genérico — substitua pela URL correta da release desejada
   wget https://github.com/solana-labs/solana/releases/download/v<SOME_TAG>/solana-release-x86_64-unknown-linux-gnu.tar.bz2
   tar -xjf solana-release-*.tar.bz2
   ./solana-release-*/install
   ```

Se nada funcionar, reporte a saída dos comandos `curl --version` e `openssl version -a`, e a saída do `openssl s_client` para ajudar na investigação.

### Boas práticas e avisos
- NUNCA use essas chaves em mainnet.
- TESTE tudo em `solana-test-validator` ou testnet antes de ir para redes públicas.

1. **Implemente verificação de saldo** antes de enviar
2. **Adicione suporte a Smart Contracts** no Ethereum
3. **Implemente EIP-1559** (base fee + priority fee)
4. **Adicione suporte a multisig** no Bitcoin
5. **Crie interface gráfica** para facilitar uso

## 🤝 Contribuindo

Este é um projeto educacional. Sinta-se livre para:
- Reportar bugs
- Sugerir melhorias
- Adicionar mais exemplos
- Melhorar documentação

## ⚠️ Avisos Importantes

1. **NUNCA use estas chaves em mainnet**
2. **NUNCA compartilhe suas chaves privadas**
3. **Este código é para ESTUDO, não para produção**
4. **Sempre teste em regtest (Bitcoin) ou testnets (Ethereum)**
5. **Faça backup das suas seeds mnemônicas**

## 📚 Referências

- [BIP39 - Mnemonic](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki)
- [BIP32 - HD Wallets](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki)
- [BIP44 - Multi-Account Hierarchy](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki)
- [BIP84 - Native SegWit](https://github.com/bitcoin/bips/blob/master/bip-0084.mediawiki)
- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
- [EIP-155 - Simple Replay Attack Protection](https://eips.ethereum.org/EIPS/eip-155)

## 📄 Licença

MIT License - Use livremente para estudos!

---

**Feito com ❤️ para estudar Bitcoin e Ethereum na prática**
