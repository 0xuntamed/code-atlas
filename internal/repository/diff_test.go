package repository

import "testing"

func TestParseUnifiedDiff(t *testing.T) {
	diff := "diff --git a/src/orders.ts b/src/orders.ts\n" +
		"index 1111111..2222222 100644\n" +
		"--- a/src/orders.ts\n" +
		"+++ b/src/orders.ts\n" +
		"@@ -10,2 +10,5 @@ export function fetchOrders() {\n" +
		"@@ -40 +43,0 @@\n" + // pure deletion, new-side count 0
		"diff --git a/src/new.ts b/src/new.ts\n" +
		"new file mode 100644\n" +
		"--- /dev/null\n" +
		"+++ b/src/new.ts\n" +
		"@@ -0,0 +1,12 @@\n" +
		"diff --git a/src/gone.ts b/src/gone.ts\n" +
		"deleted file mode 100644\n" +
		"--- a/src/gone.ts\n" +
		"+++ /dev/null\n" +
		"@@ -1,8 +0,0 @@\n"

	files := parseUnifiedDiff(diff)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	orders := files[0]
	if orders.Path != "src/orders.ts" || orders.Added || orders.Deleted {
		t.Errorf("orders file wrong: %+v", orders)
	}
	if len(orders.Ranges) != 2 || orders.Ranges[0] != (LineRange{10, 14}) || orders.Ranges[1] != (LineRange{43, 43}) {
		t.Errorf("orders ranges = %+v, want [{10 14} {43 43}]", orders.Ranges)
	}

	added := files[1]
	if added.Path != "src/new.ts" || !added.Added || added.Ranges[0] != (LineRange{1, 12}) {
		t.Errorf("added file wrong: %+v", added)
	}

	deleted := files[2]
	if deleted.Path != "src/gone.ts" || !deleted.Deleted {
		t.Errorf("deleted file wrong: %+v", deleted)
	}
}
