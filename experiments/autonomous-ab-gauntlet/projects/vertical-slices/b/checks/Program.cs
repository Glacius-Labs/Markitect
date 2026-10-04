using Commerce.Billing.Application;
using Commerce.Inventory.Application;
using Commerce.Orders.Application.PlaceOrder;
using Commerce.Orders.Domain;
var failures = new List<string>();
void Check(bool condition, string name) { if (!condition) failures.Add(name); }
var handler = new PlaceOrderHandler(new InMemoryStockAvailability(), new InMemoryPaymentAuthorizer());
var instant = new DateTimeOffset(2026, 10, 4, 12, 0, 0, TimeSpan.Zero);
var one = handler.Handle(new PlaceOrderCommand("O-1", instant, [new OrderLine("SKU-1", 1, 12.50m)], "tok_test"));
Check(one.Total == 12.50m, "one unit total");
Check(one.Status == OrderStatus.Open, "new order is open");
Check(one.Lines[0].Quantity == 1, "quantity is preserved");
var shipped = handler.Handle(new PlaceOrderCommand("O-2", instant, [new OrderLine("SKU-2", 1, 3m)], "tok_test"));
shipped.MarkShipped();
Check(shipped.Status == OrderStatus.Shipped, "open order can ship");
try { Order.Place("O-invalid", instant, [new OrderLine("SKU-1", 0, 1m)]); Check(false, "zero quantity rejected"); }
catch (ArgumentException) { Check(true, "zero quantity rejected"); }
var sourceRoot = Path.GetFullPath(Path.Combine(AppContext.BaseDirectory, "../../../../src/Commerce"));
var ordersSources = Directory.GetFiles(Path.Combine(sourceRoot, "Orders"), "*.cs", SearchOption.AllDirectories);
var forbidden = ordersSources.SelectMany(path => File.ReadAllText(path).Split('\n').Select((line, index) => (line, index, path)))
    .Where(item => item.line.Contains("Commerce.Inventory.Application", StringComparison.Ordinal) || item.line.Contains("Commerce.Inventory.Domain", StringComparison.Ordinal) || item.line.Contains("Commerce.Billing.Application", StringComparison.Ordinal) || item.line.Contains("Commerce.Billing.Domain", StringComparison.Ordinal) || item.line.Contains("Infrastructure", StringComparison.Ordinal)).ToArray();
Check(forbidden.Length == 0, "orders uses published contracts only");
if (failures.Count == 0) { Console.WriteLine("PASS: behavior and source-boundary checks (6 assertions)"); return 0; }
foreach (var failure in failures) Console.Error.WriteLine($"FAIL: {failure}");
return 1;
