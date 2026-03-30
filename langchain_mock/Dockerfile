# LangChain 契约 MOCK：与网关 downstream.langchain_http 联调（无真实推理）
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app

EXPOSE 1989

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "1989"]
