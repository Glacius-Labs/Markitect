"""Public behavioral checks against a fresh DB; destructive only to test orders."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import urllib.error
import urllib.request


class Checks:
    def __init__(self, base_url):
        self.url = base_url.rstrip("/")
        self.passed = []

    def http(self, method, route, payload=None, key=None):
        headers = {"Content-Type": "application/json"}
        if key is not None:
            headers["Idempotency-Key"] = key
        body = json.dumps(payload).encode() if payload is not None else None
        request = urllib.request.Request(self.url + route, data=body, headers=headers, method=method)
        try:
            response = urllib.request.urlopen(request, timeout=10)
        except urllib.error.HTTPError as exc:
            response = exc
        with response:
            return response.status, json.loads(response.read())

    def expect(self, name, condition):
        if not condition:
            raise AssertionError(name)
        self.passed.append(name)

    def order(self, qty=1, sku="WIDGET"):
        return {"items": [{"sku": sku, "quantity": qty}]}

    def inventory(self, sku="WIDGET"):
        status, value = self.http("GET", f"/inventory/{sku}")
        self.expect(f"inventory {sku} readable", status == 200)
        return value

    def run(self, through, condition):
        self.expect("health", self.http("GET", "/health")[0] == 200)
        status, catalog = self.http("GET", "/catalog")
        self.expect("catalog prices", status == 200 and
                    {x["sku"]: x["unitPriceCents"] for x in catalog} == {"WIDGET": 1250, "GADGET": 775})
        for name, payload, code in (
            ("zero", self.order(0), "validation"), ("negative", self.order(-1), "validation"),
            ("fractional", self.order(1.5), "validation"), ("unknown", self.order(1, "NOPE"), "unknown_sku"),
            ("empty", {"items": []}, "validation"), ("limit", self.order(11), "order_limit")):
            status, error = self.http("POST", "/orders", payload, f"invalid-{name}")
            self.expect(f"invalid {name}", status == 400 and error.get("code") == code and bool(error.get("message")))
        self.expect("missing key", self.http("POST", "/orders", self.order())[0] == 400)
        missing_status, missing_order = self.http("GET", "/orders/public-unknown-order")
        self.expect("unknown order", missing_status == 404 and missing_order.get("code") == "not_found")
        payload = {"items": [{"sku": "WIDGET", "quantity": 1}, {"sku": "GADGET", "quantity": 1},
                              {"sku": "WIDGET", "quantity": 1}]}
        status, order = self.http("POST", "/orders", payload, "public-initial")
        self.expect("created exact total", status == 201 and order["totalCents"] == 3275 and order["status"] == "accepted")
        self.expect("read order", self.http("GET", f"/orders/{order['id']}")[1] == order)
        canonical = {"items": [{"sku": "GADGET", "quantity": 1}, {"sku": "WIDGET", "quantity": 2}]}
        status, retry = self.http("POST", "/orders", canonical, "public-initial")
        self.expect("semantic retry", status == 200 and retry["id"] == order["id"])
        self.expect("key conflict", self.http("POST", "/orders", self.order(3), "public-initial")[0] == 409)
        if condition == "brownfield":
            self.expect("legacy existing path", self.http("POST", "/legacy/orders", self.order(), "public-legacy")[0] == 201)
        checkpoint = {"order": order, "inventory": None}
        if through < 2:
            return checkpoint
        inv = self.inventory()
        self.expect("reservation accounting", inv["onHand"] == 10 and inv["reserved"] == (3 if condition == "brownfield" else 2)
                    and inv["available"] == inv["onHand"] - inv["reserved"])
        with ThreadPoolExecutor(max_workers=8) as pool:
            retries = list(pool.map(lambda _: self.http("POST", "/orders", canonical, "public-initial"), range(12)))
        self.expect("concurrent duplicate same order", all(s == 200 and o["id"] == order["id"] for s, o in retries))
        self.expect("duplicate no extra reservation", self.inventory() == inv)
        available = inv["available"]
        with ThreadPoolExecutor(max_workers=8) as pool:
            attempts = list(pool.map(lambda n: self.http("POST", "/orders", self.order(), f"parallel-{n}"), range(12)))
        accepted = [o for s, o in attempts if s == 201]
        self.expect("concurrent no oversell", len(accepted) == available and
                    all(s == 201 or (s == 409 and o["code"] == "insufficient_stock") for s, o in attempts))
        full = self.inventory()
        self.expect("stock exhausted", full == {"sku": "WIDGET", "onHand": 10, "reserved": 10, "available": 0})
        self.expect("rejection atomic", self.http("POST", "/orders", self.order(), "exhausted")[0] == 409 and self.inventory() == full)
        gadget_before = self.inventory("GADGET")
        mixed = {"items": [{"sku": "GADGET", "quantity": 1}, {"sku": "WIDGET", "quantity": 1}]}
        self.expect("multi SKU all or nothing", self.http("POST", "/orders", mixed, "mixed-insufficient")[0] == 409
                    and self.inventory("GADGET") == gadget_before and self.inventory() == full)
        checkpoint["inventory"] = full
        if through < 4:
            return checkpoint
        route = f"/orders/{order['id']}/cancel"
        cancel_status, cancelled = self.http("POST", route, {})
        self.expect("cancel", cancel_status == 200 and cancelled.get("status") == "cancelled")
        after_cancel = self.inventory()
        self.expect("cancel releases once", after_cancel["reserved"] == 8 and after_cancel["available"] == 2)
        self.expect("repeat cancel", self.http("POST", route, {})[0] == 200 and self.inventory() == after_cancel)
        self.expect("cancel cannot fulfill", self.http("POST", f"/orders/{order['id']}/fulfill", {})[0] == 409)
        target = accepted[0]["id"]
        fulfill_status, fulfilled_order = self.http("POST", f"/orders/{target}/fulfill", {})
        self.expect("fulfill", fulfill_status == 200 and fulfilled_order.get("status") == "fulfilled")
        fulfilled = self.inventory()
        self.expect("fulfill consumes once", fulfilled["onHand"] == 9 and fulfilled["reserved"] == 7 and fulfilled["available"] == 2)
        self.expect("repeat fulfill", self.http("POST", f"/orders/{target}/fulfill", {})[0] == 200 and self.inventory() == fulfilled)
        self.expect("fulfilled cannot cancel", self.http("POST", f"/orders/{target}/cancel", {})[0] == 409)
        checkpoint["order"] = self.http("GET", f"/orders/{order['id']}")[1]
        checkpoint["inventory"] = fulfilled
        if through >= 5:
            for route in ("/orders", "/legacy/orders"):
                status, error = self.http("POST", route, self.order(7), "new-limit-" + route)
                self.expect(f"new limit {route}", status == 400 and error.get("code") == "order_limit")
            self.expect("new boundary valid", self.http("POST", "/orders", self.order(6, "GADGET"), "boundary-six")[0] == 201)
            self.expect("legacy neighbor valid", self.http("POST", "/legacy/orders", self.order(), "legacy-neighbor")[0] == 201)
            checkpoint["inventory"] = self.inventory()
        return checkpoint

    def restart(self, checkpoint):
        self.expect("order durable after restart", self.http("GET", f"/orders/{checkpoint['order']['id']}")[1] == checkpoint["order"])
        if checkpoint["inventory"] is not None:
            self.expect("inventory durable after restart", self.inventory() == checkpoint["inventory"])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--through-task", type=int, choices=range(1, 7), default=1)
    parser.add_argument("--condition", choices=("greenfield", "brownfield"), required=True)
    parser.add_argument("--checkpoint")
    parser.add_argument("--verify-restart")
    args = parser.parse_args()
    checks = Checks(args.base_url)
    try:
        if args.verify_restart:
            checks.restart(json.loads(Path(args.verify_restart).read_text(encoding="utf-8")))
        else:
            checkpoint = checks.run(args.through_task, args.condition)
            if args.checkpoint:
                Path(args.checkpoint).write_text(json.dumps(checkpoint, indent=2) + "\n", encoding="utf-8")
        print(json.dumps({"status": "passed", "checks": checks.passed}, indent=2))
    except (AssertionError, OSError, KeyError, ValueError) as exc:
        print(json.dumps({"status": "failed", "passed": checks.passed, "error": str(exc)}, indent=2))
        raise SystemExit(1)
