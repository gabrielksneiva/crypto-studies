// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../contracts/MyToken.sol";

/**
 * Script de deploy do token ERC-20
 * 
 * Este é um "forge script" - a forma moderna e recomendada de fazer deploys
 * com o Foundry. Ele oferece mais controle e confiabilidade que o "forge create".
 * 
 * Como funciona:
 * 1. A função run() é executada automaticamente
 * 2. vm.broadcast() faz a próxima transação ser enviada para a blockchain
 * 3. O construtor MyToken() é chamado com o supply inicial
 * 4. O endereço do contrato é logado no console
 */
contract DeployScript is Script {
    function run() external {
        // Pega o supply inicial dos argumentos (padrão: 1 milhão)
        uint256 initialSupply = vm.envOr("INITIAL_SUPPLY", uint256(1000000));
        
        // Inicia o broadcast (próximas transações serão enviadas)
        vm.startBroadcast();
        
        // Deploy do contrato
        MyToken token = new MyToken(initialSupply);
        
        // Para o broadcast
        vm.stopBroadcast();
        
        // Log do endereço para fácil referência
        console.log("Token deployed at:", address(token));
        console.log("Initial supply:", initialSupply);
        console.log("Deployer:", msg.sender);
    }
}
