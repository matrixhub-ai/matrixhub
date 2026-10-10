FROM python:3.12-alpine
RUN pip install --no-cache-dir fickling==0.1.12
WORKDIR /app
COPY analyze.py worker.py clamd-security.conf /app/
RUN mkdir /spool && chown 65534:65534 /spool && chmod 700 /spool
USER 65534:65534
ENV PYTHONDONTWRITEBYTECODE=1
CMD ["python", "-B", "/app/worker.py"]
