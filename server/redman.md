# 🛒 Sistema de Gestão Operacional (SGO)

> Uma solução de baixo custo para gestão de escalas e controle de insumos em supermercados, otimizada para hardware de recursos limitados.

## 📄 Sobre o Projeto

Este projeto nasceu da necessidade de organizar processos internos recorrentes em um supermercado, especificamente a **Escala Diária de Funcionários** e a **Retirada de Materiais** (limpeza e escritório).

O objetivo central foi criar uma "nova forma de fazer as coisas", substituindo controles manuais e despadronizados por um sistema digital eficiente, sem exigir investimento em infraestrutura por parte da empresa.

## 🎯 Desafio de Engenharia: Otimização e Custo Zero

A arquitetura foi desenhada com restrições estritas de hardware e orçamento para garantir que a implantação tivesse custo zero para o estabelecimento, reaproveitando equipamentos existentes e dispositivos de baixo consumo.

### 🖥️ Compatibilidade Legada (Client-Side)
O Frontend foi desenvolvido e otimizado especificamente para garantir compatibilidade total e performance fluida em **navegadores Firefox antigos**, que compõem o parque tecnológico atual dos terminais da empresa.

### ⚙️ Servidor em Edge Computing (Server-Side)
Para eliminar a necessidade de servidores dedicados caros ou custos mensais de nuvem, o backend foi projetado para rodar em uma **TV Box** adaptada.

**Especificações do Ambiente de Produção:**
* **Hardware:** TV Box Genérica.
* **Arquitetura:** Processador ARMv7 (32-bit).
* **Memória:** 2GB de RAM.
* **Armazenamento/OS:** Linux rodando via Cartão SD (Boot externo).

Esta abordagem prova que é possível entregar valor de negócio e modernização digital utilizando recursos computacionais mínimos (Low-End Hardware).

## 🚀 Funcionalidades Principais

* **Escala Diária:** Visualização e gestão dos turnos dos colaboradores.
* **Controle de Materiais:** Registro digital de retirada de insumos (evitando desperdícios e falhas de controle).
* **Interface Leve:** Design focado em usabilidade e baixo consumo de memória do navegador.

## 🛠️ Tecnologias Utilizadas

* **Linguagem:** Java
* **Backend:** Spring Boot (Otimizado para baixo consumo de memória)
* **Frontend:** JSP (JavaServer Pages) - Escolhido pela compatibilidade e renderização server-side.
* **Sistema Operacional:** Linux (Distribuição leve para ARM)


codigo para compila: GOOS=linux GOARCH=arm GOARM=7 go build -o app main.go 

---
*Desenvolvido por [Seu Nome] - Focado em Engenharia de Software sob Restrições.*