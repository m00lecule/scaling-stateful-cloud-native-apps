import logging
import random
import time
import uuid

import requests

from locust import HttpUser, between, events, task

TESTS_ID = uuid.uuid4()
  
PRODUCTS_DATA = [
    {"name": f"p01-{TESTS_ID}", "stock": 100},
    {"name": f"p02-{TESTS_ID}", "stock": 20},
    {"name": f"p03-{TESTS_ID}", "stock": 50},
    {"name": f"p04-{TESTS_ID}", "stock": 30}
]

logging.basicConfig(level=logging.INFO)

PRODUCTS_STOCKS = {}

MAX_PRODUCTS = 1
MAX_COUNT = 12

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    host = environment.host
    print(f"STARTING STATEFUL-APP STRESS TESTS - {TESTS_ID}")
    print("STARTING SEEDING DATA")
    for p in PRODUCTS_DATA:
        print("processing", p["name"])
        r = requests.post(url = f"{host}/products", json = p)
        print("response", r.status_code)
        data = r.json()
        PRODUCTS_STOCKS[data["payload"]["ID"]] = data["payload"]["Stock"]
    print("COMPLETED DATA SEEDING")


@events.quitting.add_listener
def _(environment, **_kwargs):
    print("WILL REMOVE SEEDED DATA")
    host = environment.host
    for p in PRODUCTS_STOCKS.keys():
        print("processing", p)
        r = requests.delete(url = f"{host}/products/{p}")
        print("response", r.status_code)
    print("REMOVED SEEDED DATA")

class AppUser(HttpUser):
    wait_time = between(1, 5)

    @task
    def cart_order_test(self):
        print("test")
        response = self.client.post("/carts/")
        session_id = response.json()["payload"]

        products = random.randint(1, 2)
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

            response = self.client.patch(f"/carts/{session_id}", name="/carts", json=data)

            print("Adding product response status code:", response.status_code)

            time.sleep(10)

        response = self.client.post(f"/carts/{session_id}/submit")
        print("Submit response status code:", response.status_code)

        print("orders done: ", session_id, " ", orders_done)

        while True:
            time.sleep(1)
