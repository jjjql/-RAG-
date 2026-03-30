FROM python:3.11-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    RAG_AI_BACKEND=fastembed

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app
COPY tests ./tests

# Linux 容器内默认 UDS；Windows 宿主机联调可改用 TCP + 端口映射
ENV RAG_AI_TRANSPORT=unix
ENV RAG_AI_SOCKET_PATH=/tmp/rag_gateway.sock

EXPOSE 18080

CMD ["python", "-m", "app.main", "--unix-path", "/tmp/rag_gateway.sock"]
