import logging
import os
import random
import time
import uuid
import json

import requests

from locust import HttpUser, between, events, task

from locust.clients import HttpSession

TESTS_ID = uuid.uuid4()

INF_WAIT = int(os.getenv("INF_WAIT", 0))
PRODUCTS_NO = int(os.getenv("PRODUCT_NO", 200))
SEQUENCES = int(os.getenv("SEQUENCES", 1))

PRODUCTS_DATA = [ {"name": f"p{p}-{TESTS_ID}", "stock": 10000000000000} for p in range(PRODUCTS_NO) ]

logging.basicConfig(level=logging.INFO)

PRODUCTS_STOCKS = {}

MAX_PRODUCTS = 1
MAX_COUNT = 12

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    host = environment.host
    logging.info("STARTING STATEFUL-APP STRESS TESTS - %s", TESTS_ID)
    logging.info("STARTING SEEDING DATA")
    for p in PRODUCTS_DATA:
        logging.info("processing %s", p["name"])
        r = requests.post(url = f"{host}/products", json = p)
        logging.info("response %d", r.status_code)
        data = r.json()
        PRODUCTS_STOCKS[data["payload"]["ID"]] = data["payload"]["Stock"]
    logging.info("COMPLETED DATA SEEDING")


@events.quitting.add_listener
def _(environment, **_kwargs):
    logging.info("WILL REMOVE SEEDED DATA")
    host = environment.host
    for p in PRODUCTS_STOCKS.keys():
        logging.info("processing %s", p)
        r = requests.delete(url = f"{host}/products/{p}")
        logging.info("response %d", r.status_code)
    logging.info("REMOVED SEEDED DATA")

class AppUser(HttpUser):
    wait_time = between(1, 5)

    @task
    def cart_order_test(self):
        for _ in range(SEQUENCES):
            response = self.client.post("/carts/")
            cart_id = response.json()["payload"]

            products = random.randint(2, 4)
            orders_done = {}

            for i in range(products):
                product_id = random.choice(list(PRODUCTS_STOCKS.keys()))
                product_count = random.randint(1, 10)

                data = {
                    "details" : {
                        product_id: product_count
                    }
                }

                count = orders_done.get(product_id, 0)

                orders_done[product_id] = count + product_count

                print(product_id, orders_done[product_id])

                response = self.client.patch(f"/carts/{cart_id}", name="/carts", json=data)

                logging.info("Order resp code: %d", response.status_code)

                response = self.client.get(f"/carts/{cart_id}")

                curr_cart = response.json()["payload"]["Content"]
                
                for k,v in orders_done.items():
                    assert v == curr_cart[str(k)]["Count"]               

                time.sleep(3)

            response = self.client.post(f"/carts/{cart_id}/submit")
            logging.info("Submit resp code: %s", response.status_code)

            logging.info("ordered %s - %s", cart_id, json.dumps(orders_done))
            self.client.cookies.clear()

        while INF_WAIT:
            time.sleep(1)
