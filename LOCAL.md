# Ambiente local isolado

Este ambiente usa dois bancos novos e separados no PostgreSQL do computador,
com o usuario do Odoo. Nao copia nem altera os dados do servidor ou do Odoo.
A aplicacao tem acesso de saida a internet para conectar ao WhatsApp. Um proxy
separado publica o acesso apenas no computador local. A conexao automatica das
instancias esta desativada; use uma linha de testes separada da producao.
Esta versao nao solicita registro de licenca; as condicoes de licenca e
atribuicao dos projetos continuam aplicaveis.

Depois de uma queda, sessoes pareadas usam a reconexao nativa do whatsmeow,
com espera progressiva entre tentativas, sem recriar o cliente a cada evento
`Disconnected`. A biblioteca tambem recupera conexoes sem resposta de keepalive
(limite atual de tres minutos desde a ultima resposta). Os eventos
`KeepAliveTimeout` e `KeepAliveRestored` pertencem a inscricao `CONNECTION`.
Isso nao ativa a conexao automatica ao iniciar o processo, nem substitui um novo
pareamento quando a sessao foi encerrada no celular. A reconexao manual e os
fluxos de parada ainda precisam da revisao de ciclo de vida prevista no checklist.

O Manager conta contatos sincronizados pelo WhatsApp enquanto a instancia esta
conectada. A contagem inclui mensagens recebidas e envios pelos endpoints
`/send`, exceto status, observados a partir da ativacao desta funcionalidade,
sem importar historico.
Ela depende de `DATABASE_SAVE_MESSAGES=true`, registra apenas identificadores
por instancia e pode levar alguns segundos para atualizar. Conversas ainda nao
possuem contagem disponivel. Erros de persistencia sao registrados nos logs.

O codigo-fonte do Manager fica em `manager/`. O Docker compila essa versao
tanto para o servico Manager quanto para a interface incluida na imagem da API.
O historico Git anterior do Manager esta guardado localmente em
`.git/manager-history`; ele nao e incluido em novos clones.

O arquivo `.env` na raiz do repositorio deve ter permissoes restritas,
`LOCAL_API_KEY`, `LOCAL_DB_USER` e `LOCAL_DB_PASSWORD`. Atualmente, o usuario e
a senha sao os do PostgreSQL local usado pelo Odoo. Caracteres especiais no
usuario ou na senha do banco exigem codificacao na URL de conexao. Nao
compartilhe esse arquivo.

O endereco padrao do PostgreSQL a partir da rede Docker e `172.18.0.1`.
Caso o gateway da rede mude, consulte-o com:

```sh
docker network inspect evolution-go-local_local --format '{{range .IPAM.Config}}{{.Gateway}}{{end}}'
```

Defina `LOCAL_DB_HOST` no `.env` com o gateway exibido.
Depois de alterar qualquer variavel no `.env`, aplique a configuracao com:

```sh
docker compose -f docker-compose.local.yml up -d
```

Para criar e iniciar os servicos:

```sh
docker compose -f docker-compose.local.yml up -d --build
docker compose -f docker-compose.local.yml logs evolution-go
```

Neste computador, abra `http://127.0.0.1:18080/manager/login` e use o valor de
`LOCAL_API_KEY` como chave de API. Para parar sem apagar o banco:

```sh
docker compose -f docker-compose.local.yml stop
```

Nao restaure dados de producao neste ambiente sem antes revisar sessoes,
webhooks, integracoes e compatibilidade da versao. O historico e as integracoes
do Chatwoot tambem dependem dos dados e da configuracao do proprio Chatwoot.
