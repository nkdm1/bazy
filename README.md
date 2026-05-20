# please - read this slowly and carefully 
### table of contents
- [download the repo](#download-the-repo)
- [linux vm: setup the database environment](#linux-vm:-setup-the-database-environment)
- [your pc: setup the app environment](#your-pc:-setup-the-app-environment)
- [docker commands](#docker-commands)
- [resources](#resources)
## download the repo
### on your pc and a linux vm:
1. install [git](https://git-scm.com/install/)
2. clone the repo: ` git clone https://github.com/nkdm1/bazy.git `
3. cd to repo: `cd bazy`
4. **IMPORTANT** get familiar with how the repo looks like by running: `tree -d -L 2`

## linux vm: setup the database environment 
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
    sudo docker-compose up -d 
    ```
    or `sudo docker compose up -d` - depends on your linux distro
2. check if everything is ok: `sudo docker-compose ls`

## your pc: setup the app environment
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
4. edit .env file so that DB_ADDR variable has assigned the correct IP - **don't change anything else**
### start development loop
1. start air in a **seperate** terminal window:
    ```
    cd app/cmd
    go tool air
    ```
    air will recompile and restart the application on any file change
3. send curl requests to the application - for example to check the database status:
    `curl localhost:8080/status`

## docker commands

* start the database: `sudo docker compose up -d`
* stop the database: `sudo docker compose down`
* stop and reset the database: `sudo docker compose down -v`
* launches interactive mariadb shell as user: `sudo docker compose exec db mariadb -u user -puser db`
* launches interactive mariadb shell as root: `sudo docker compose exec db mariadb -u root -proot db` 

## resources
* git:
    * https://youtu.be/Q62uJjPHF3U -> me yapping about the git workflow
    * https://www.youtube.com/watch?v=Ala6PHlYjmw -> understand what is happening behind the commands - watch until 6:30


