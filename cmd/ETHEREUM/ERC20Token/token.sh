#!/bin/bash

# ========== SCRIPT DE DEPLOY E INTERAÇÃO COM ERC-20 ==========
# Analogia: Este script é como um gerenciador de banco que automatiza
# a criação da moeda e todas as operações (enviar, aprovar, consultar)

set -e

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configurações padrão
RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"
PRIVATE_KEY="${PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}" # Anvil account #0
CONTRACT_FILE="contracts/MyToken.sol:MyToken"
BUILD_DIR="out"

# ========== FUNÇÕES AUXILIARES ==========

print_header() {
    echo -e "${BLUE}========== $1 ==========${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# ========== VERIFICAR DEPENDÊNCIAS ==========

check_dependencies() {
    print_header "VERIFICANDO DEPENDÊNCIAS"
    
    if ! command -v forge &> /dev/null; then
        print_error "Foundry não instalado!"
        echo "Execute: curl -L https://foundry.paradigm.xyz | bash && foundryup"
        exit 1
    fi
    
    print_success "Foundry instalado"
    
    # Verificar se o RPC está rodando
    if ! curl -s -X POST -H "Content-Type: application/json" \
         --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
         $RPC_URL > /dev/null 2>&1; then
        print_error "Nó Ethereum não está rodando em $RPC_URL"
        echo "Execute: anvil"
        exit 1
    fi
    
    print_success "Nó Ethereum rodando"
    echo
}

# ========== DEPLOY DO CONTRATO ==========

deploy() {
    print_header "DEPLOY DO CONTRATO ERC-20"
    
    local initial_supply=${1:-1000000}
    
    print_info "Supply inicial: $initial_supply tokens"
    print_info "Compilando contrato..."
    
    # Compilar contrato
    forge build
    
    print_success "Contrato compilado"
    print_info "Fazendo deploy..."
    
    # Deploy (com --broadcast para fazer deploy real, --legacy para compatibilidade)
    output=$(forge create $CONTRACT_FILE \
        --rpc-url $RPC_URL \
        --private-key $PRIVATE_KEY \
        --constructor-args $initial_supply \
        --legacy \
        --broadcast \
        2>&1)
    
    # Extrair endereço do contrato
    contract_address=$(echo "$output" | grep "Deployed to:" | awk '{print $3}')
    
    if [ -z "$contract_address" ]; then
        print_error "Falha no deploy"
        echo "$output"
        exit 1
    fi
    
    print_success "Contrato deployed!"
    echo -e "${GREEN}Endereço: $contract_address${NC}"
    echo
    echo "Salve este endereço para usar nos próximos comandos:"
    echo "export CONTRACT=$contract_address"
    echo
    
    # Salvar em arquivo
    echo "$contract_address" > .contract_address
    print_info "Endereço salvo em .contract_address"
}

# ========== VERIFICAR SALDO ==========

balance() {
    print_header "CONSULTAR SALDO"
    
    local contract=${1:-$(cat .contract_address 2>/dev/null)}
    local account=${2:-$(cast wallet address $PRIVATE_KEY)}
    
    if [ -z "$contract" ]; then
        print_error "Endereço do contrato não fornecido"
        echo "Use: $0 balance <contract_address> [account_address]"
        exit 1
    fi
    
    echo "Contrato: $contract"
    echo "Conta: $account"
    echo
    
    # Consultar saldo (balanceOf)
    balance_raw=$(cast call $contract \
        "balanceOf(address)(uint256)" \
        $account \
        --rpc-url $RPC_URL)
    
    # Converter de wei para tokens (dividir por 10^18)
    balance_tokens=$(cast --from-wei $balance_raw)
    
    print_success "Saldo: $balance_tokens MLT"
    echo "  (raw: $balance_raw wei)"
}

# ========== TRANSFERIR TOKENS ==========

transfer() {
    print_header "TRANSFERIR TOKENS"
    
    local contract=${1:-$(cat .contract_address 2>/dev/null)}
    local to=$2
    local amount=$3
    
    if [ -z "$contract" ] || [ -z "$to" ] || [ -z "$amount" ]; then
        print_error "Parâmetros insuficientes"
        echo "Use: $0 transfer <contract> <to_address> <amount>"
        exit 1
    fi
    
    # Converter tokens para wei (multiplicar por 10^18)
    amount_wei=$(cast --to-wei $amount)
    
    echo "De: $(cast wallet address $PRIVATE_KEY)"
    echo "Para: $to"
    echo "Quantidade: $amount tokens ($amount_wei wei)"
    echo
    
    print_info "Enviando transação..."
    
    # Executar transfer
    tx_hash=$(cast send $contract \
        "transfer(address,uint256)(bool)" \
        $to \
        $amount_wei \
        --rpc-url $RPC_URL \
        --private-key $PRIVATE_KEY \
        --json | jq -r '.transactionHash')
    
    print_success "Transação enviada!"
    echo "TX Hash: $tx_hash"
    
    # Aguardar confirmação
    print_info "Aguardando confirmação..."
    cast receipt $tx_hash --rpc-url $RPC_URL > /dev/null
    
    print_success "Confirmado!"
}

# ========== APROVAR TOKENS ==========

approve() {
    print_header "APROVAR TOKENS"
    
    local contract=${1:-$(cat .contract_address 2>/dev/null)}
    local spender=$2
    local amount=$3
    
    if [ -z "$contract" ] || [ -z "$spender" ] || [ -z "$amount" ]; then
        print_error "Parâmetros insuficientes"
        echo "Use: $0 approve <contract> <spender_address> <amount>"
        exit 1
    fi
    
    amount_wei=$(cast --to-wei $amount)
    
    echo "Owner: $(cast wallet address $PRIVATE_KEY)"
    echo "Spender: $spender"
    echo "Quantidade: $amount tokens"
    echo
    
    print_info "Enviando transação..."
    
    tx_hash=$(cast send $contract \
        "approve(address,uint256)(bool)" \
        $spender \
        $amount_wei \
        --rpc-url $RPC_URL \
        --private-key $PRIVATE_KEY \
        --json | jq -r '.transactionHash')
    
    print_success "Approve enviado!"
    echo "TX Hash: $tx_hash"
}

# ========== VERIFICAR ALLOWANCE ==========

allowance() {
    print_header "CONSULTAR ALLOWANCE"
    
    local contract=${1:-$(cat .contract_address 2>/dev/null)}
    local owner=${2:-$(cast wallet address $PRIVATE_KEY)}
    local spender=$3
    
    if [ -z "$contract" ] || [ -z "$spender" ]; then
        print_error "Parâmetros insuficientes"
        echo "Use: $0 allowance <contract> [owner_address] <spender_address>"
        exit 1
    fi
    
    echo "Owner: $owner"
    echo "Spender: $spender"
    echo
    
    allowance_raw=$(cast call $contract \
        "allowance(address,address)(uint256)" \
        $owner \
        $spender \
        --rpc-url $RPC_URL)
    
    allowance_tokens=$(cast --from-wei $allowance_raw)
    
    print_success "Allowance: $allowance_tokens MLT"
}

# ========== INFO DO TOKEN ==========

info() {
    print_header "INFORMAÇÕES DO TOKEN"
    
    local contract=${1:-$(cat .contract_address 2>/dev/null)}
    
    if [ -z "$contract" ]; then
        print_error "Endereço do contrato não fornecido"
        exit 1
    fi
    
    echo "Contrato: $contract"
    echo
    
    # Nome
    name=$(cast call $contract "name()(string)" --rpc-url $RPC_URL)
    print_info "Nome: $name"
    
    # Símbolo
    symbol=$(cast call $contract "symbol()(string)" --rpc-url $RPC_URL)
    print_info "Símbolo: $symbol"
    
    # Total Supply
    total_supply_raw=$(cast call $contract "totalSupply()(uint256)" --rpc-url $RPC_URL)
    total_supply=$(cast --from-wei $total_supply_raw)
    print_info "Total Supply: $total_supply $symbol"
}

# ========== MENU PRINCIPAL ==========

show_help() {
    cat << EOF
${BLUE}========== ERC-20 TOKEN MANAGER ==========${NC}

Uso: $0 <comando> [argumentos]

${GREEN}Comandos disponíveis:${NC}

  ${YELLOW}deploy [supply]${NC}
    Faz deploy de um novo contrato ERC-20
    Exemplo: $0 deploy 1000000

  ${YELLOW}info [contract]${NC}
    Mostra informações do token
    Exemplo: $0 info 0x123...

  ${YELLOW}balance [contract] [account]${NC}
    Consulta saldo de uma conta
    Exemplo: $0 balance 0x123... 0xabc...

  ${YELLOW}transfer <contract> <to> <amount>${NC}
    Transfere tokens para outra conta
    Exemplo: $0 transfer 0x123... 0xabc... 100

  ${YELLOW}approve <contract> <spender> <amount>${NC}
    Aprova um endereço a gastar seus tokens
    Exemplo: $0 approve 0x123... 0xdef... 50

  ${YELLOW}allowance <contract> [owner] <spender>${NC}
    Verifica quanto foi aprovado
    Exemplo: $0 allowance 0x123... 0xabc... 0xdef...

${GREEN}Variáveis de ambiente:${NC}

  RPC_URL      Endpoint do nó (padrão: http://127.0.0.1:8545)
  PRIVATE_KEY  Chave privada (padrão: Anvil account #0)

${GREEN}Exemplo de fluxo completo:${NC}

  1. Iniciar Anvil:
     ${YELLOW}anvil${NC}

  2. Deploy do token:
     ${YELLOW}$0 deploy 1000000${NC}

  3. Verificar saldo:
     ${YELLOW}$0 balance${NC}

  4. Transferir para conta #1 do Anvil:
     ${YELLOW}$0 transfer \$CONTRACT 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 100${NC}

  5. Aprovar conta #1 a gastar 50 tokens:
     ${YELLOW}$0 approve \$CONTRACT 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 50${NC}

EOF
}

# ========== MAIN ==========

main() {
    local command=$1
    shift || true
    
    case "$command" in
        deploy)
            check_dependencies
            deploy "$@"
            ;;
        info)
            check_dependencies
            info "$@"
            ;;
        balance)
            check_dependencies
            balance "$@"
            ;;
        transfer)
            check_dependencies
            transfer "$@"
            ;;
        approve)
            check_dependencies
            approve "$@"
            ;;
        allowance)
            check_dependencies
            allowance "$@"
            ;;
        help|--help|-h|"")
            show_help
            ;;
        *)
            print_error "Comando desconhecido: $command"
            echo
            show_help
            exit 1
            ;;
    esac
}

main "$@"
