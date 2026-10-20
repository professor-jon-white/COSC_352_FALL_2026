Programming Language
For this project, I decided to use Go (Golang)

Datasets
The application uses two Baltimore community-level datasets:
data/crime.csv — Violent crime rates per 1,000 residents in 2023.
data/poverty.csv — Household poverty percentages in 2023.
Both datasets use the CSA2020 column to identify Baltimore Community Statistical Areas. The program joins records using this shared column.

Research Questions
Which Baltimore community has the highest violent crime rate?
Which Baltimore community has the highest household poverty percentage?
What are the average violent crime rate and household poverty percentage across Baltimore communities?
Which communities have both violent crime rates and poverty percentages above their respective averages?
Which communities have violent crime rates above average but poverty percentages below average?

Concurrency
The application loads the two CSV datasets concurrently using goroutines and channels.
It also uses a worker pool with three goroutines to process community records. A sync.WaitGroup makes sure that all workers finish before the program collects the results.

Design Pattern
The application uses a Strategy design pattern primarily. The RankingStrategy interface allows the application to rank communities using different metrics. CrimeStrategy ranks communities by violent crime rates, while PovertyStrategy ranks communities by poverty percentages.

Visualizations
The program displays terminal-based bar charts that show the five communities with the highest violent crime rates and the five communities with the highest household poverty percentages.
It also displays ranked community lists and numerical summaries.

How to Install on macOS devices
1) Install Go 
Download and install Go from https://go.dev/dl/ and make sure you pick the correct one for your computer
2) Open Terminal
Navigate to the project directory
3) Make sure Go is installed
In the Terminal, run “go version”
4) Run the application
In the Terminal, run “go run .”
5) Compile the application
In the Terminal, Run “go build -o baltimore-city-insight”
6) Run the compiled application
In the Terminal, run “./baltimore-city-insight”
(MAKE SURE YOU RUN THESE COMMANDS FROM THE PROJECT DIRECTORY SO THE APPLICATION CAN LOCATE THE CSV FILES THAT ARE IN THE DATA FOLDER.)
