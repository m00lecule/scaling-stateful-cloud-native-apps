import time
from locust import HttpUser, task, between

class AppUser(HttpUser):
    wait_time = between(1, 5)

    @task
    def cart_test(self):
        response = self.client.post("/carts/")
        print(response.json())

        while True:
            time.sleep(1)
