# consumer-service
consumer server for substrate-cli. intermediatory server between llm-node and api-server. does not have any endpoints, only communicates via rabbitmq channel.

# environment variables

```bash
ANTHROPIC_KEY=
ANTHROPIC_MAX_TOKENS=32000
ANTHROPIC_MAX_TOKENS_PRECHECK=1024
API_SERVER_URL="http://localhost:8080"
GEMINI_API_KEY=
OPENAI_KEY=
OPENAI_MAX_TOKENS=32000
OPENAI_MAX_TOKENS_PRECHECK=1024
SAFE_ORIGINS="http://localhost:8090, http://localhost:8080, http://localhost:3000"
PORT=8090
NODE="consumer-service"
MODE="cli"
DEFAULT_MODEL="anthropic"
AMQP_URL="amqp://guest:guest@localhost:5672/"
REDIS_ADDR="localhost:6379"
BUNDLE="server"
SUPPORTED_MODELS="anthropic,openai,gemini"
```

# run consumer-service

```bash
go run ./cmd/app
```

this is an entry point to substrate-cli, for more informations, follow instructions on https://trysubstrate.com/notes.     
api-server - https://github.com/substrate-cli/api-server.     
llm-node - https://github.com/substrate-cli/llm-node-cli.     
