from app.services import accounts


@app.get("/python/accounts")
def list_accounts():
    return accounts.list_all()
