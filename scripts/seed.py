import logging

import requests

URL = "http://127.0.0.1:8080/products/"
  
products = [
    {"name": "p01", "stock": 100},
    {"name": "p02", "stock": 20},
    {"name": "p03", "stock": 50},
    {"name": "p04", "stock": 30}
]

logging.basicConfig(level=logging.INFO)

for p in products:
    logging.info("processing %s", p["name"])
    r = requests.post(url = URL, json = p)
    logging.info("response %d", r.status_code)
