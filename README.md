AquillesJudger
========

Descrição
---------
Uma API Judger (executor/avaliador) para programação competitiva, pensado para a aplicação aquilles.run.

Fluxo:
- Recebe requisição com: UUID do problema, linguagem e código.
- Busca os dados de teste no cache local ou baixa da API (ambos configurados via `.env`).
- Cada problema no cache é um diretório contendo `n.in`, `n.out` e `meta.json`.
- Cria uma pasta temporária de execução com arquivos de entrada/saída e limites, então inicia um container Docker que executa o binário `runner` para rodar e comparar.
- Ao terminar de rodar, envia um post com os dados para 

Arquitetura (visão geral)
- API: expõe endpoints que recebem as requisições de execução (pasta `internal/api` / `cmd`).
- Cache service: gerencia os problemas baixados e os armazena em `internal/api/cache/`.
- Worker / Runner: o worker organiza a criação da execução e invoca um binário runner (gerenciado em `pkg/runner` e empacotado em `internal/api/binaries/runner`).
- Modularidade: o código do worker é modularizado em `pkg/` (veja `pkg/config`, `pkg/runner`, `pkg/worker`).

Formato do cache / API
----------------------
A API (remote) retorna um arquivo ZIP com a estrutura esperada para um problema:

- `1.in`, `1.out`, `2.in`, `2.out`, ...
- `meta.json`

Exemplo:

- `1.in` -> "hello world"
- `1.out` -> "hello world"
- `2.in` -> "oi"
- `2.out` -> "oi"
- `meta.json` ->

	[{"language":"python","time_limit":42,"memory_limit":23}]

Observações sobre `meta.json`:
- `time_limit`: segundos
- `memory_limit`: megabytes

Comportamento do cache
----------------------
- Diretório padrão: `./internal/api/cache/` (configurável via `.env` `CACHE_DIRECTORY`).
- Cada problema é armazenado em um diretório com sufixo `-problem` (também configurável `CACHE_FILEEXTENSION`) (ex.: `81995a45-...-problem`) contendo os arquivos de casos de teste e `meta.json`.
- Quando o serviço precisa do problema, ele verifica o cache local; se não existir e `ONLY_LOCAL_CACHE=false`, baixa o ZIP da API e descompacta no `CACHE_DIRECTORY`.
- Se `ONLY_LOCAL_CACHE=true`, o serviço só usa o conteúdo local do cache (útil para ambientes off-line ou testes).

Como popular o cache manualmente
-------------------------------
Crie um diretório dentro de `internal/api/cache/` com o nome do UUID seguido do sufixo configurado no .env e coloque os arquivos `n.in`, `n.out` e `meta.json` lá. Exemplo:

internal/api/cache/UUIDTeste-problem/
- 1.in
- 1.out
- meta.json

Configuração (.env)
-------------------

.env:
- `API_URL`: endpoint para baixar o .zip do problema.
- `API_CALLBACK_URL`: URL de callback para reportar resultados.
- `API_KEY`: token para autenticar requisições a API remota.
- `CACHE_DIRECTORY`: onde os problemas são armazenados localmente.
- `CACHE_FILEEXTENSION`: sufixo padrão usado ao armazenar (ex.: `-problem`).
- `EXECUTION_DIRECTORY`: pasta para execuções temporárias (geralmente dentro do cache: `.../executions`).
- `RUNNER_BINARY_PATH`: caminho para o binário runner usado dentro do container (ex.: `./internal/api/binaries/runner`).
- `CONTAINER_TIMEOUT_SECONDS`, `MAX_WORKERS`, `QUEUE_SIZE` e `ONLY_LOCAL_CACHE` controlam limites e comportamento do serviço.

Note: consulte `.env.example` para valores padrão. Se quiser rodar só em local, mantenha `ONLY_LOCAL_CACHE=true` e popule manualmente o cache.

Executando localmente
---------------------
Pré-requisitos:
- Go 1.20+ (ou compatível)
- Docker (ou outro runtime compatível) em execução

Construir o runner (gera o binário usado pelo container):

```bash
go build -o internal/api/binaries/runner ./pkg/runner
```

Build / Run da API:

```bash
# rodar sem build: (desenvolvimento)
go run ./cmd/main.go

# ou criar binário do servidor
go build -o bin/ifjudger ./cmd
./bin/ifjudger
```

Notas sobre Docker e runner
--------------------------
- O serviço cria containers Docker para isolar execuções; portanto, o Docker daemon precisa estar disponível ao usuário que roda o serviço.
- O binário `runner` é invocado dentro do container, o código-fonte está em `pkg/runner`.

Estrutura relevante do projeto
-----------------------------
- `cmd/` — entrada da aplicação (`main.go`).
- `internal/api/` — controllers, serviços, cache e binários usados pela API.
- `pkg/` — código modular reutilizável: `pkg/worker`, `pkg/runner`, `pkg/config`.

Exemplos rápidos de uso
----------------------

Compilar runner e iniciar serviço:

```bash
go build -o internal/api/binaries/runner ./pkg/runner
go run ./cmd/main.go
```

Segurança e limites
-------------------
- Evite rodar o serviço com privilégios desnecessários. O isolamento por container reduz o risco, mas atenção ao montar volumes e ao tempo de execução configurado em `CONTAINER_TIMEOUT_SECONDS`.


Linguagens suportadas
---------------------
- `python` — atualmente suportado. O runner/worker usa imagens Docker para isolar execuções; portanto a imagem `python:3.12.12-slim` do Python precisa estar disponível no host.

Certifique-se de ter a imagem presente (ou faça pull):

```bash
docker pull python:3.12.12-slim
```
