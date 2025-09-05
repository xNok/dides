


run-postgres:
	docker run --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=mysecretpassword -d postgres 

kill-postgres:
	docker stop postgres
	docker rm postgres