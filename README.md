# Peek
Compile instructions

Pre-requisites
1. You will need CMake, Golang, Redis installed to run the program. Program talks to redis on port 6379
2. The webservice is written in Golang
3. The webservice is running by default on port3000. It can be changed by giving the "-p=<port>" on the executable

Compile procedure
1. cd server
2. mkdir build
3. cd build
4. cmake ..
5. make
6. ./httpserver

Data model

In redis, there are a bunch of keys and sets created for handling the bookings

a. Timestamp keys, start with "ts:" are timestampids storing timestamp information in a json
b. There is a dated set (eg. 2014-07-22) that holds all timestamp keys for a particular day
c. Boat keys, start with "boat:" are boatids storing boat information in a json
d. There is a set name boats, which contain all the boat ids created
e. There is a set starting with "asmt:", which stores the association of a boatid to the many associated timestampids

Caveats due to time constraint

1. Have not put any logging information
2. Have not considered all corner and validation scenarios
