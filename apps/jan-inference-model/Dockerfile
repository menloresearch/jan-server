FROM python:3.12-slim

WORKDIR /app
COPY application/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY application/mockserver.py app.py
EXPOSE 8101
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8101"]