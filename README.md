## install git
on your pc and a linux vm you should:
1. install [git](https://git-scm.com/install/)
2. clone the repo: ` git clone https://github.com/nkdm1/bazy.git `

## setup the database environment (linux vm)
### first installation
1. connect to your linux vm
2. install:
    - [docker engine](https://docs.docker.com/engine/install/)
    - [docker compose plugin](https://docs.docker.com/compose/install/linux/)
3. enable the docker daemon: `sudo systemctl enable --now docker`
### wake up the database
1. cd to db/dev directory and start docker container:
    ```
    cd db/dev/
    sudo docker compose up -d 
    ```
2. database should be available via port 3306

## setup the app environment (your pc)
### first installation
1. install [go](https://go.dev/doc/install)
2. cd to app/ directory and install go modules:
    ```
    cd app
    go mod tidy
    ```
3. create .env with database credentials and address:
    ```
    touch cmd/.env
    cat << EOF > cmd/.env
    DB_ADDR=<YOUR-LINUX-VM-IP>:3306
    DB_USER=user
    DB_PASSWORD=user
    EOF
    ```
4. change the DB_ADDR variable so it contains the correct IP
### start development loop
1. start air in a **seperate** terminal window:
    ```
    cd app/cmd
    go tool air
    ```
2. air will recompile and start the application on any file change
3. send curl requests to the application - for example to check the database status:
    `curl localhost:8080/status`

## docker commands:
- sudo docker compose up -d 
  - starts the database server

- sudo docker compose down
  - stops the database server

- sudo docker compose down -v  
  - stops the database server and 
    unmounts the volume so all the data from the 
    database tables will be wiped out on the next start

- sudo docker compose exec db mariadb -u user -puser db 
  - launches interactive mariadb shell as user

- sudo docker compose exec db mariadb -u root -proot db 
  - launches interactive mariadb shell as root 


