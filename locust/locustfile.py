import random

from locust import FastHttpUser, task, between


def product(product_id):
    return {
        "product_id": product_id,
        "sku": "ABC-123-XYZ",
        "manufacturer": "Acme Corporation",
        "category_id": 456,
        "weight": 1250,
        "some_other_id": 789,
    }


class ProductUser(FastHttpUser):
    abstract = True
    wait_time = between(0.1, 0.5)

    def on_start(self):
        # Create one product so GETs have something to find
        self.product_ids = []
        self.post_product()

    def get_product(self):
        product_id = random.choice(self.product_ids)
        self.client.get(f"/products/{product_id}", name="/products/{productId}")

    def post_product(self):
        # IDs well away from the ones the README and Postman examples use (12345, 99999)
        product_id = random.randint(1_000_000, 2_000_000)
        with self.client.post(
            f"/products/{product_id}/details",
            json=product(product_id),
            name="/products/{productId}/details",
            catch_response=True,
        ) as response:
            if response.status_code == 204:
                self.product_ids.append(product_id)
                response.success()
            else:
                response.failure(f"expected 204, got {response.status_code}")


# GET:POST ratios to choose from in the web UI (--class-picker)
class ReadHeavy3to1(ProductUser):
    tasks = {ProductUser.get_product: 3, ProductUser.post_product: 1}


class Balanced1to1(ProductUser):
    tasks = {ProductUser.get_product: 1, ProductUser.post_product: 1}


class WriteHeavy1to3(ProductUser):
    tasks = {ProductUser.get_product: 1, ProductUser.post_product: 3}
