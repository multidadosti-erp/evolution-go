# Manager

Interface React e TypeScript integrada ao Evolution GO, baseada no
[Evolution Manager v2](https://github.com/evolution-foundation/evolution-manager-v2)
da Evolution API Team. A licenca e as condicoes do projeto original estao em
[LICENSE](LICENSE).

Para executar o ambiente local e compilar os containers, consulte
[LOCAL.md](../LOCAL.md). O Dockerfile nesta pasta compila o servico Manager;
o Dockerfile na raiz tambem compila a interface para inclui-la na imagem da API.

Para desenvolvimento da interface, use `npm ci` e `npm run dev` nesta pasta.
Para verificar a compilacao, use `npm run build`.
