## how to setup local database
1. install docker and git on a linux vm

2. 'git clone https://github.com/nkdm1/bazy.git' to the linux vm

3. CREATE db/.env FILE AND PASTE:
   - MARIADB_ROOT_PASSWORD="USERPASSWORD"
   - MARIADB_PASSWORD="ROOTPASSWORD"

docker commands:
- sudo docker compose up -d 
  - starts the database server

- sudo docker compose down
  - stops the database server

- sudo docker compose down -v  
  - stops the database server and 
    unmounts the volume so all the data from the 
    database tables will be wiped out on the next start

- sudo docker compose exec database mariadb -u user -pUSERPASSWORD db 
  - launches interactive mariadb shell as user

- sudo docker compose exec database mariadb -u root -pROOTPASSWORD db 
  - launches interactive mariadb shell as root 


