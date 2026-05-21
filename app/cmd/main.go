package main


func main() {
	api := &api{
		db: databaseConnect(),
	}
	api.run(api.mount())
}
