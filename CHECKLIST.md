# Checklist de evolução do Evolution GO

Objetivo: evoluir e validar a versão própria localmente antes de substituir a
instalação de produção. Marcar um item apenas após sua validação.

## Retomada

Em 13/11/2026, a correção de QR foi aplicada à API local. Os testes isolados
passaram e três consultas de QR com sessão autenticada mantiveram a conexão.
Após o reinício autorizado, a linha foi reconectada e suas configurações foram
preservadas. A produção não foi alterada.

Em 16/11/2026, foi habilitada no código a recuperação nativa do whatsmeow, em vez
de recriar o cliente em cada evento de desconexão. Foram adicionados os eventos
de keepalive à inscrição CONNECTION. Testes isolados, testes com detector de
corridas e compilação da API passaram. Com autorização, o ambiente local foi
construído e iniciado. Um bloqueio seletivo do destino WhatsApp provocou timeout
de keepalive; a sessão se recuperou automaticamente em cerca de 210 segundos,
sem reiniciar o processo nem recriar o cliente. Configurações preservadas e
bloqueio removido; nenhuma mensagem de teste foi enviada.

A primeira tentativa de simulação, desconectando a rede externa Docker, também
afetou o acesso ao banco e foi interrompida com restauração da rede. Usar bloqueio
seletivo do destino WhatsApp nos próximos testes para não afetar o PostgreSQL.

As alterações do repositório ainda não foram commitadas. O ambiente local está
ativo e a linha está conectada. Próximo passo: revisar a serialização das
operações manuais de ciclo de vida e validar envio e recebimento após recuperação,
com autorização para troca de mensagens. Concluir também os testes reais de novo
QR e pareamento por código quando houver linha disponível.

## 1. Base local

- [x] Clonar o Evolution GO e preparar ambiente local isolado.
- [x] Incorporar o código-fonte do Manager ao repositório.
- [x] Conectar uma linha de teste e validar envio e recebimento de mensagens.
- [x] Ajustar o dashboard para contatos e contagem de mensagens observadas.
- [ ] Revisar as alterações locais existentes e organizar seu versionamento.
- [ ] Comparar a imagem personalizada de produção com a versão clonada e
      identificar funcionalidades ou configurações que precisam ser mantidas.

## 2. Estabilidade e reconexão

- [ ] Levantar logs e métricas do travamento em produção, sem expor credenciais
      ou conteúdo de mensagens; identificar a causa antes de atribuí-la a um PR.
- [x] Corrigir a consulta de QR para não reiniciar uma sessão já logada e
      validar com testes isolados o início de cliente, QR existente e passkey.
      Referência: [PR #149](https://github.com/evolution-foundation/evolution-go/pull/149).
- [x] Validar em execução que consultas de QR não interrompem a sessão existente.
- [ ] Validar em execução a geração de um novo QR e o pareamento por código,
      usando uma linha de teste disponível.
- [x] Avaliar recuperação de conexão sem resposta e habilitar a reconexão
      nativa com backoff, removendo a recriação automática de clientes no evento
      Disconnected; validar eventos e inscrições com testes isolados.
      Referências: [#160](https://github.com/evolution-foundation/evolution-go/pull/160),
      [#197](https://github.com/evolution-foundation/evolution-go/pull/197).
- [x] Validar recuperação de keepalive com sessão real e bloqueio seletivo de
      rede, mantendo processo, cliente e configurações.
- [ ] Avaliar a serialização do ciclo de vida por instância e a sobreposição
      entre os PRs antes de adaptar correções.
      Referências: [#234](https://github.com/evolution-foundation/evolution-go/pull/234),
      [#239](https://github.com/evolution-foundation/evolution-go/pull/239),
      [#240](https://github.com/evolution-foundation/evolution-go/pull/240),
      [#244](https://github.com/evolution-foundation/evolution-go/pull/244).
- [ ] Testar queda e retorno de rede, reconexão concorrente e reinício,
      verificando sessão, recebimento, envio e ausência de clientes duplicados.
- [ ] Definir política de conexão automática na inicialização; atualmente está
      desativada no ambiente local.
- [ ] Melhorar logs e monitoramento para distinguir processo ativo de sessão
      WhatsApp efetivamente conectada.

## 3. Mensagens e eventos

- [ ] Preservar configurações e inscrições de eventos no fluxo de desconexão e
      reconexão; testar também os webhooks.
      Referência: [PR #187](https://github.com/evolution-foundation/evolution-go/pull/187).
- [ ] Avaliar destinatários de mensagens enviadas em dispositivos vinculados;
      testar conversas individuais, grupos e identificadores JID/LID.
      Referência: [PR #191](https://github.com/evolution-foundation/evolution-go/pull/191).
- [ ] Revisar normalização de números brasileiros e testar diferentes DDDs,
      formatos atuais e legados.
      Referência: [PR #208](https://github.com/evolution-foundation/evolution-go/pull/208).
- [ ] Avaliar correção da solicitação de histórico e testar o comportamento
      real, sem presumir importação completa.
      Referência: [PR #218](https://github.com/evolution-foundation/evolution-go/pull/218).
- [ ] Validar envio e recebimento de texto, mídia e mensagens de dispositivos
      vinculados, incluindo repetição de eventos e falhas temporárias.

## 4. Integração com Chatwoot

- [ ] Definir arquitetura, configuração, requisitos e fluxo bidirecional da
      integração antes de implementá-la.
- [ ] Implementar e testar entrega de mensagens e eventos do WhatsApp ao
      Chatwoot e respostas do Chatwoot ao WhatsApp.
- [ ] Tratar identificação de contatos e conversas, duplicidade, retentativas
      e prevenção de loops.
- [ ] Definir e testar tratamento de mídia, grupos e histórico conforme os
      requisitos escolhidos.
- [ ] Disponibilizar configuração no Manager após a API e a integração estarem
      implementadas e testadas.

## 5. Manager, segurança e operação

- [ ] Revisar limitações do dashboard: a contagem começa na ativação da
      funcionalidade, depende da persistência de mensagens e não inclui histórico;
      contagem de conversas ainda não está disponível.
- [ ] Analisar vulnerabilidades informadas pelo npm e atualizar dependências
      com testes de compatibilidade.
- [ ] Revisar credenciais, permissões, exposição de portas e proteção dos
      webhooks; substituir credenciais expostas antes da implantação.
- [ ] Definir backup e testar restauração isolada de bancos, volumes e
      configurações, sem conectar simultaneamente a sessão de produção.
- [ ] Documentar configuração, atualização, diagnóstico e recuperação.

## 6. Preparação para produção

- [ ] Definir versão própria e processo reproduzível de build da imagem.
- [ ] Validar em ambiente de homologação os fluxos necessários e a estabilidade
      após interrupções, incluindo testes de longa duração.
- [ ] Preparar compose e configurações de produção sem credenciais no
      repositório; confirmar compatibilidade com bancos e dados existentes.
- [ ] Definir janela, backup verificado, procedimento de implantação e plano de
      retorno para a imagem anterior.
- [ ] Implantar somente com autorização explícita e acompanhar logs,
      conexão e troca de mensagens após a mudança.

## Critérios de trabalho

- Adaptar e testar correções dos PRs individualmente; não aplicar pacotes
  amplos sem revisão e sem considerar as alterações locais.
- Não alterar produção nem interromper a linha local sem autorização.
- Não registrar credenciais, números ou conteúdo de mensagens neste arquivo.
