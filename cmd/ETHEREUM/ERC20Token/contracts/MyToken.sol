// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// ========== O QUE É UM TOKEN ERC-20? ==========
// Analogia: É como criar sua própria moeda dentro do Ethereum.
// Assim como o Real Brasileiro (BRL) ou Dólar (USD), mas digital e programável.
//
// Exemplos reais: USDT, USDC, UNI, LINK, MATIC
//
// O padrão ERC-20 define 6 funções obrigatórias:
//   1. totalSupply() - quanto da moeda existe no total
//   2. balanceOf(address) - quanto uma pessoa tem
//   3. transfer(to, amount) - enviar para alguém
//   4. approve(spender, amount) - autorizar alguém a gastar por você
//   5. allowance(owner, spender) - ver quanto foi autorizado
//   6. transferFrom(from, to, amount) - gastar tokens autorizados
//
// E 3 eventos (logs) obrigatórios:
//   1. Transfer - quando tokens são enviados
//   2. Approval - quando alguém é autorizado

contract MyToken {
    // ========== PASSO 1: DADOS BÁSICOS DO TOKEN ==========
    // Analogia: É como o nome, símbolo e casas decimais da moeda.
    // Exemplo: "Dólar Americano" = USD, 2 casas decimais ($1.00)
    
    string public name = "My Learning Token";
    string public symbol = "MLT";
    uint8 public decimals = 18; // Padrão Ethereum (1 token = 10^18 wei)
    
    uint256 public totalSupply; // Total de tokens criados
    
    // Analogia: Livro-caixa que registra quanto cada pessoa tem
    mapping(address => uint256) public balanceOf;
    
    // Analogia: Procurações - quem pode gastar em nome de quem e quanto
    // allowance[dono][gastador] = quanto o gastador pode usar
    mapping(address => mapping(address => uint256)) public allowance;
    
    // ========== EVENTOS (LOGS) ==========
    // Analogia: Extrato bancário - registra todas as movimentações
    
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    
    // ========== PASSO 2: CRIAR O SUPPLY INICIAL ==========
    // Analogia: O governo criando a moeda pela primeira vez
    // Aqui o criador do contrato recebe todos os tokens iniciais
    
    constructor(uint256 initialSupply) {
        totalSupply = initialSupply * 10**decimals; // Converte para wei
        balanceOf[msg.sender] = totalSupply; // Criador recebe tudo
        
        emit Transfer(address(0), msg.sender, totalSupply);
    }
    
    // ========== PASSO 3: TRANSFERIR TOKENS ==========
    // Analogia: Você enviando dinheiro para outra pessoa via Pix
    // Você só pode enviar o que você tem
    
    function transfer(address to, uint256 amount) public returns (bool) {
        // Validações
        require(to != address(0), "Nao pode enviar para endereco zero");
        require(balanceOf[msg.sender] >= amount, "Saldo insuficiente");
        
        // Atualizar saldos (débito e crédito)
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        
        // Registrar no extrato
        emit Transfer(msg.sender, to, amount);
        
        return true;
    }
    
    // ========== PASSO 4: APROVAR TERCEIROS (ALLOWANCE) ==========
    // Analogia: Você assina uma procuração autorizando alguém a usar seu dinheiro
    // Exemplo: autorizar Uniswap a trocar seus tokens por você
    
    function approve(address spender, uint256 amount) public returns (bool) {
        require(spender != address(0), "Nao pode aprovar endereco zero");
        
        // Registrar a aprovação
        allowance[msg.sender][spender] = amount;
        
        emit Approval(msg.sender, spender, amount);
        
        return true;
    }
    
    // ========== PASSO 5: TRANSFERIR EM NOME DE ALGUÉM ==========
    // Analogia: Usar a procuração para movimentar dinheiro de outra pessoa
    // Exemplo: Uniswap pegando seus tokens aprovados para fazer a troca
    
    function transferFrom(address from, address to, uint256 amount) public returns (bool) {
        require(from != address(0), "From nao pode ser zero");
        require(to != address(0), "To nao pode ser zero");
        require(balanceOf[from] >= amount, "Saldo insuficiente do remetente");
        require(allowance[from][msg.sender] >= amount, "Allowance insuficiente");
        
        // Atualizar saldos
        balanceOf[from] -= amount;
        balanceOf[to] += amount;
        
        // Reduzir o allowance usado
        allowance[from][msg.sender] -= amount;
        
        // Registrar no extrato
        emit Transfer(from, to, amount);
        
        return true;
    }
    
    // ========== FUNÇÕES EXTRAS (OPCIONAIS) ==========
    
    // Queimar tokens (destruir permanentemente)
    // Analogia: Tirar dinheiro de circulação (Banco Central faz isso)
    function burn(uint256 amount) public returns (bool) {
        require(balanceOf[msg.sender] >= amount, "Saldo insuficiente para queimar");
        
        balanceOf[msg.sender] -= amount;
        totalSupply -= amount;
        
        emit Transfer(msg.sender, address(0), amount);
        
        return true;
    }
    
    // Criar mais tokens (mint)
    // Analogia: Imprimir mais dinheiro (só o dono pode fazer)
    address public owner;
    
    modifier onlyOwner() {
        require(msg.sender == owner, "Apenas o dono pode fazer isso");
        _;
    }
    
    function mint(address to, uint256 amount) public onlyOwner returns (bool) {
        require(to != address(0), "Nao pode criar para endereco zero");
        
        totalSupply += amount;
        balanceOf[to] += amount;
        
        emit Transfer(address(0), to, amount);
        
        return true;
    }
}
