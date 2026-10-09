# Prueba técnica Leasity

Este repositorio contiene mi entrega de la prueba técnica realizada por la empresa Leasity para la posición de Backend Software Engineer. Consiste en una integración con un proveedor de pagos externo escrita en Go. Puedes visitar la versión de producción [aquí](http://ec2-3-91-160-165.compute-1.amazonaws.com/).

## Cómo levantar (escrito con IA 🤖):

Para levantar el proyecto en local necesitas:

- [Docker](https://docs.docker.com/get-docker/) y Docker Compose (opción recomendada), **o bien**
- Go 1.27.2 o superior y un PostgreSQL accesible.
- Un `credentials.json` de una cuenta de servicio de Google con acceso a Google Sheets (ver más abajo).

### Variables de entorno

Crea un archivo `.env` en la raíz del proyecto. Docker Compose lo lee automáticamente para completar las variables.

```dotenv
DB_HOST=localhost
DB_PORT=5432
DB_USER=user
DB_PASSWORD=password
DB_NAME=database

GOOGLE_CREDENTIALS_FILE=credentials.json
SPREADSHEET_ID=tu_spreadsheet_id
SHEET_NAME=Hoja 1
```

| Variable | Descripción |
| --- | --- |
| `DB_HOST` | Host de PostgreSQL. Con Docker Compose se sobreescribe a `database`; en local usa `localhost`. |
| `DB_PORT` | Puerto de PostgreSQL (por defecto `5432`). |
| `DB_USER` | Usuario de PostgreSQL. |
| `DB_PASSWORD` | Contraseña de PostgreSQL. |
| `DB_NAME` | Nombre de la base de datos. |
| `GOOGLE_CREDENTIALS_FILE` | Ruta al JSON de la cuenta de servicio (ej. `credentials.json`). |
| `SPREADSHEET_ID` | ID de la planilla de Google Sheets (está en la URL de la planilla). |
| `SHEET_NAME` | Nombre exacto de la pestaña dentro de la planilla (ej. `Hoja 1`). |

> `.env` y `credentials.json` están en `.gitignore`: no se versionan. No los subas al repositorio.

### Credenciales de Google Sheets

1. En [Google Cloud Console](https://console.cloud.google.com/), habilitá la **Google Sheets API** en el proyecto.
2. Creá una **cuenta de servicio** y generá una clave en formato JSON.
3. Guardá el archivo como `credentials.json` en la raíz del proyecto.
4. Compartí la planilla con el email de la cuenta de servicio (`client_email` del JSON) con permiso de **Editor**.
5. Copiá el ID de la planilla (la parte entre `/d/` y `/edit` en la URL) en `SPREADSHEET_ID`.

### Levantar con Docker (recomendado)

```bash
docker compose up --build
```

Esto levanta dos contenedores:

- `leasity-hw`: la API, disponible en `http://localhost`.
- `postgres_db`: PostgreSQL, disponible en `localhost:5432`.

Los datos de PostgreSQL se conservan en el volumen `postgres_data`.

Para detener:

```bash
docker compose down        # detiene los contenedores y conserva los datos
docker compose down -v     # además borra el volumen de PostgreSQL
```

## Stack:

Decidí escoger librerías relativamente simples, pero que solucionaran las cosas que no quiero implementar de cero. Para todo el manejo de requests y endpoints de la API escogí Gin, y para conectar con mi base de datos relacional (más sobre esta decisión abajo) ocupé GORM.

## Decisiones de arquitectura:

Tomé un par de decisiones de alto nivel que vale la pena mencionar; tanto en sus beneficios como debilidades:

- *Uso de base de datos relacional como fuente de verdad:* Escogí Postgres para guardar los datos. Es simple y conocido, y las garantías ACID permiten asegurar un estado consistente de la base de datos. Asímismo, esta decisión encaja perfectamente con tres de los requisitos del proyecto, a saber: (i) el proveedor puede enviar el mismo evento más de una vez, (ii) el proveedor puede actualizar el estado de un pago, y (iii) los eventos pueden llegar fuera de orden. Utilizando key constraints podemos asegurar trivialmente la idempotencia del endpoint POST, sin tener que implementarlo a nivel de la API; funciona a nivel de la base de datos.

- *Consistencia de data entre la base de datos y la planilla:* El sistema está diseñado de manera tal que, para cada escritura correcta a la base de datos, se reescriba desde cero la planilla en Google Sheets. Esto trae como consecuencia el hecho de que puede haber un corto periodo de desfase entre la base de datos y la planilla, pero el sistema es eventualmente consistente, lo cual considero que es suficiente. Generalmente, planillas como Google Sheets son utilizados con fines contables y no como fuente de verdad, por lo que me parece que el posible desfase de ~1 segundo es tolerable. Con esta decisión, obtenemos todos los beneficios de performance, simplicidad y consistencia que ofrece un DBMS, y se ahorra el costo de desarrollar y mantener todo un sistema de escritura y lectura de Google Sheets.

No obstante lo anterior, en un sistema con una cantidad muy grande de pagos, valdría la pena considerar cambiar este flujo, que será el principal cuello de botella. Probablemente existan librerías que logren escribir de manera eficiente una tabla relacional a Google Sheets.

## Consideraciones para escalar:

Para escalar un sistema como este, hay que tener en consideración el tipo de tráfico que rercibe: en escala, la mayoría de solicitudes deberían ser solicitudes POST de pagos/eventos. En este caso, para mejorar el performance del sistema (si fuese necesario), pondría la mayor cantidad de esfuerzo posible en hacer más eficiente la escritura a Google Sheets, que no está optimizado para performance. Dicho eso, si hubiesen problemas de rendimiento en la base de datos (probablemente por escritura), también es posible hacer sharding, utilizando como sharding key algo como el hash del payment_id.
