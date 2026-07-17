import { listUsers } from "../services/users.js";

export function auditRequest() {
  return listUsers();
}

router.get("/users", authenticate, listUsers);
