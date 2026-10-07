using System.Text.Json;
using Microsoft.Data.Sqlite;

namespace Orders.Api;

public sealed class OrdersStore(string databasePath)
{
    private const int OrderLimit = 10;
    private readonly string _connectionString = new SqliteConnectionStringBuilder
    {
        DataSource = Path.GetFullPath(databasePath),
        Mode = SqliteOpenMode.ReadWriteCreate
    }.ToString();

    public void Initialize()
    {
        using var connection = Open();
        using var command = connection.CreateCommand();
        command.CommandText = """
            CREATE TABLE IF NOT EXISTS catalog (
                sku TEXT PRIMARY KEY COLLATE BINARY,
                unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
                on_hand INTEGER NOT NULL CHECK (on_hand >= 0));
            CREATE TABLE IF NOT EXISTS orders (
                id TEXT PRIMARY KEY,
                items_json TEXT NOT NULL,
                total_cents INTEGER NOT NULL,
                status TEXT NOT NULL);
            CREATE TABLE IF NOT EXISTS idempotency (
                key TEXT PRIMARY KEY,
                fingerprint TEXT NOT NULL,
                order_id TEXT NOT NULL REFERENCES orders(id));
            INSERT OR IGNORE INTO catalog (sku, unit_price_cents, on_hand)
                VALUES ('WIDGET', 1250, 10), ('GADGET', 775, 6);
            """;
        command.ExecuteNonQuery();
    }

    public IReadOnlyList<CatalogItem> GetCatalog()
    {
        using var connection = Open();
        using var command = connection.CreateCommand();
        command.CommandText = "SELECT sku, unit_price_cents FROM catalog ORDER BY sku COLLATE BINARY";
        using var reader = command.ExecuteReader();
        var items = new List<CatalogItem>();
        while (reader.Read())
            items.Add(new CatalogItem(reader.GetString(0), reader.GetInt32(1)));
        return items;
    }

    public InventoryItem? GetInventory(string sku)
    {
        using var connection = Open();
        using var command = connection.CreateCommand();
        command.CommandText = "SELECT sku, on_hand FROM catalog WHERE sku = $sku COLLATE BINARY";
        command.Parameters.AddWithValue("$sku", sku);
        using var reader = command.ExecuteReader();
        return reader.Read() ? new InventoryItem(reader.GetString(0), reader.GetInt32(1), 0, reader.GetInt32(1)) : null;
    }

    public CreateOrderResult CreateOrder(string key, IReadOnlyList<OrderLineRequest> requestedItems)
    {
        if (requestedItems.Any(item => item is null || string.IsNullOrEmpty(item.Sku) || item.Quantity <= 0 || decimal.Truncate(item.Quantity) != item.Quantity))
            return CreateOrderResult.Failure("validation", "Each item needs a SKU and a positive integer quantity.");
        if (requestedItems.Sum(item => item.Quantity) > OrderLimit)
            return CreateOrderResult.Failure("order_limit", $"Combined order quantity cannot exceed {OrderLimit}.");

        var quantities = requestedItems
            .GroupBy(item => item.Sku, StringComparer.Ordinal)
            .ToDictionary(group => group.Key, group => group.Sum(item => checked((long)item.Quantity)), StringComparer.Ordinal);
        var fingerprint = JsonSerializer.Serialize(quantities.OrderBy(item => item.Key, StringComparer.Ordinal));
        using var connection = Open();
        using var transaction = connection.BeginTransaction();

        using (var existing = connection.CreateCommand())
        {
            existing.Transaction = transaction;
            existing.CommandText = "SELECT fingerprint, order_id FROM idempotency WHERE key = $key";
            existing.Parameters.AddWithValue("$key", key);
            using var reader = existing.ExecuteReader();
            if (reader.Read())
            {
                var sameRequest = reader.GetString(0) == fingerprint;
                var orderId = reader.GetString(1);
                reader.Close();
                transaction.Commit();
                return sameRequest
                    ? CreateOrderResult.Replay(GetOrder(connection, orderId)!)
                    : CreateOrderResult.Failure("idempotency_conflict", "This idempotency key was used for a different request.");
            }
        }

        var lines = new List<OrderLine>();
        long total = 0;
        foreach (var (sku, quantity) in quantities)
        {
            using var lookup = connection.CreateCommand();
            lookup.Transaction = transaction;
            lookup.CommandText = "SELECT unit_price_cents FROM catalog WHERE sku = $sku COLLATE BINARY";
            lookup.Parameters.AddWithValue("$sku", sku);
            var price = lookup.ExecuteScalar();
            if (price is null)
            {
                transaction.Rollback();
                return CreateOrderResult.Failure("unknown_sku", $"Unknown SKU '{sku}'.");
            }
            var unitPrice = Convert.ToInt32(price);
            total = checked(total + unitPrice * quantity);
            lines.Add(new OrderLine(sku, checked((int)quantity), unitPrice));
        }

        var order = new Order(Guid.NewGuid().ToString("N"), lines, total, "accepted");
        using (var insert = connection.CreateCommand())
        {
            insert.Transaction = transaction;
            insert.CommandText = "INSERT INTO orders (id, items_json, total_cents, status) VALUES ($id, $items, $total, $status)";
            insert.Parameters.AddWithValue("$id", order.Id);
            insert.Parameters.AddWithValue("$items", JsonSerializer.Serialize(order.Items));
            insert.Parameters.AddWithValue("$total", order.TotalCents);
            insert.Parameters.AddWithValue("$status", order.Status);
            insert.ExecuteNonQuery();
        }
        using (var insertKey = connection.CreateCommand())
        {
            insertKey.Transaction = transaction;
            insertKey.CommandText = "INSERT INTO idempotency (key, fingerprint, order_id) VALUES ($key, $fingerprint, $id)";
            insertKey.Parameters.AddWithValue("$key", key);
            insertKey.Parameters.AddWithValue("$fingerprint", fingerprint);
            insertKey.Parameters.AddWithValue("$id", order.Id);
            insertKey.ExecuteNonQuery();
        }
        transaction.Commit();
        return CreateOrderResult.Created(order);
    }

    public Order? GetOrder(string id)
    {
        using var connection = Open();
        return GetOrder(connection, id);
    }

    private static Order? GetOrder(SqliteConnection connection, string id)
    {
        using var command = connection.CreateCommand();
        command.CommandText = "SELECT items_json, total_cents, status FROM orders WHERE id = $id";
        command.Parameters.AddWithValue("$id", id);
        using var reader = command.ExecuteReader();
        return reader.Read()
            ? new Order(id, JsonSerializer.Deserialize<List<OrderLine>>(reader.GetString(0))!, reader.GetInt64(1), reader.GetString(2))
            : null;
    }

    private SqliteConnection Open()
    {
        var connection = new SqliteConnection(_connectionString);
        connection.Open();
        return connection;
    }
}
