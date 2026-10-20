Jayden Reyes

This is my first time using Golang. I tried to keep everying basic and easy to read. I have mainly used python when coding programs so I decided to use Golang since it closer to python and can be structured
similarly to python code. 

I used gemini and chatGPT to help restructure me code when needed and keep the code structure the way I wanted it to be.  

Purpose:
The purpose of this project to show and compare the stats of arrest, gun offender crimes, and community violence intervention records.

By uploading csv files from Open Baltimore, fill in the datasets for my program by using the csv file. There was a lot of trial and error building this progam. Fortunately I was able to learn how to build a 
harness for my program. Using command to check for error and test run my project was simple since I had a similary experience from my cyber security class. 

In this program you will see the dataset for arrests and the dataset for gun offender. The CVI data shows information for any activities or community meeting that are recorded in the Community Violece 
Intervention Ecosystem.   

How to Start the Program:
1. First type this command: cd baltimore-engine 
2. Then type: go run .
-The second step will run the program so that it show you all the information from the datasets with needing go in to each file individually. 

The files that I created: 
main.go: I first created the main.go file to start the program active all the functions for the datasets. 

revealer.go: The revealer.go file goes through all of the csv files and converts them for the program. The convertions happens due to the Go structs. The purpsose of struct is to create a group that 
variables can go into. An example would be variable for the type of crime, the neighborhood where it happened, as well as name of the offender would all go into on group. While other variables for events 
mediates for the Community Violence Intervention would go into another group. 

concurrency.go: The concurrency.go file uses goroutines and channels to load all of the data concurrently. I remember learning about goroutines in class and I learned about channels from a tutorial I watched
about using Golang for the first time. 

analytics.go: The analytics.go is where all of the data is processed and the comparisons between the data sets are made. 

data/: In data/ you can find the csv files that were used for this program. 

Makefile: Makefile is the instruction manual that gives you a list of commands and tells you what each of them do.

For the files: concurrency.go, revealer.go, and analytics.go, I decided to give them these names because they are easy for me to remember and explain.   

Experience: 
There were a lot of challenges I faced when I was building this program. The most difficult challenged I faced was learning a new programing language. There was a lot of trial and error when building this 
program. In the past, I usually build larger programs with a team. Doing this on my own was a really difficult challenge as well. 
