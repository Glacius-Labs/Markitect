var root = Path.Combine(Directory.GetCurrentDirectory(), "docs", "represented");
if (!Directory.Exists(root))
    return Fail("the explicitly selected docs/represented target root is missing");

var pages = Directory.EnumerateFiles(root, "*.md", SearchOption.AllDirectories).ToArray();
if (pages.Length == 0)
    return Fail("the explicitly selected docs/represented target root contains no Markdown pages");

var text = string.Join(Environment.NewLine, pages.Select(File.ReadAllText));
var required = new[]
{
    "OrderRules",
    "BillingQuery",
    "ProductComposition",
    "TryCalculateTotalCents",
    "SumOutstandingCents",
    "quantity",
    "unit price",
    "outstanding",
};

var missing = required.Where(term => !text.Contains(term, StringComparison.OrdinalIgnoreCase)).ToArray();
if (missing.Length > 0)
    return Fail("represented Markdown is missing required selected-model terms: " + string.Join(", ", missing));

Console.WriteLine($"Checked {pages.Length} Markdown page(s) under the explicitly selected target root.");
return 0;

static int Fail(string message)
{
    Console.Error.WriteLine(message);
    return 1;
}
