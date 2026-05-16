install docker and git on a linux vm
'git clone https://github.com/nkdm1/bazy.git' to the linux vm

docker compose up -d 
    starts the database server

docker compose down
    stops the database server

docker compose down -v  
    stops the database server and 
    unmounts the volume so all the data from the 
    database tables will be wiped out on the next start

sudo docker dompose exec database mariadb -u user -ppasswd db 
    launches interactive mariadb shell as user

sudo docker dompose exec database mariadb -u root -proot db 
    launches interactive mariadb shell as root 

