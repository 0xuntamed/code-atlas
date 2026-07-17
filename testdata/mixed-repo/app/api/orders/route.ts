import { fetchOrders } from "../../../src/orders";

export async function GET() {
  return fetchOrders();
}
