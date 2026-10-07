package integration

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestComposedInvoiceMatchesBothAreas(t *testing.T) {
	quantityBytes, err := os.ReadFile("../quantity/quantity.txt")
	if err != nil {
		t.Fatal(err)
	}
	priceBytes, err := os.ReadFile("../price/price.txt")
	if err != nil {
		t.Fatal(err)
	}
	invoiceBytes, err := os.ReadFile("invoice.txt")
	if err != nil {
		t.Fatal(err)
	}
	quantity, err := parseValue(string(quantityBytes), "units=")
	if err != nil {
		t.Fatal(err)
	}
	price, err := parseValue(string(priceBytes), "unit=")
	if err != nil {
		t.Fatal(err)
	}
	total, err := parseValue(string(invoiceBytes), "total=")
	if err != nil {
		t.Fatal(err)
	}
	if total != quantity*price {
		t.Fatalf("composed invariant failed: total %d does not equal units %d * unit price %d", total, quantity, price)
	}
}

func parseValue(content, prefix string) (int, error) {
	line := strings.TrimSpace(content)
	if !strings.HasPrefix(line, prefix) {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(strings.TrimPrefix(line, prefix))
}
