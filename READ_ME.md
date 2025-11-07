um judger ainda em desenvolvimento
atualmente funciona só para rodar código em python, colocar input e retornar o stdout e stderr

rota POST /worker
body:
{
  "code": string,
  "input": string
}

response:
{
    "stdout": string,
    "stderr": string,
    "error": string
}
