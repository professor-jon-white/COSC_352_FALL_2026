#!/usr/bin/env python3
"""Generates SYNTHETIC sample CSVs that mimic the Open Baltimore column layout.
Only for offline testing; use the real downloads for the actual analysis."""
import csv, random
from datetime import datetime, timedelta
random.seed(7)
N = ["Downtown","Harlem Park","Sandtown-Winchester","Belair-Edison","Canton","Fells Point","Hampden","Brooklyn",
     "Cherry Hill","Pigtown","Mount Vernon","Waverly","Frankford","Park Heights","Highlandtown","Remington",
     "Edmondson Village","Patterson Park","Roland Park","Medfield"]
W = [9,8,10,7,3,3,2,6,6,4,3,4,5,8,4,2,5,3,1,1]
CR = ["LARCENY","COMMON ASSAULT","AGG. ASSAULT","ROBBERY - STREET","BURGLARY","AUTO THEFT","SHOOTING","HOMICIDE","RAPE","ARSON"]
CW = [30,20,10,8,10,12,3,1,1,1]
SR = ["HCD-Rat Rodent Complaint","SW-Dirty Alley","SW-Cleaning","TRM-Pothole","TRM-Street Light Out","HCD-Vacant Building","SW-Bulk Pickup","TRM-Traffic Signal","WW-Water Main"]
random.seed(11)
def when(y=2025): return datetime(y,1,1)+timedelta(minutes=random.randrange(0,60*24*360))
with open("data/sample/crime_sample.csv","w",newline="") as f:
    w=csv.writer(f); w.writerow(["RowID","CrimeDateTime","Location","Neighborhood","Description","Weapon","District"])
    for i in range(6000):
        n=random.choices(N,W)[0]; d=when(2024)
        w.writerow([i,d.strftime("%Y/%m/%d %H:%M:%S+00"),"X",n,random.choices(CR,CW)[0],"NA","CENTRAL"])
    w.writerow([9999,"bad-date","X","","LARCENY","NA","X"])  # skipped row
with open("data/sample/311_sample.csv","w",newline="") as f:
    w=csv.writer(f); w.writerow(["ServiceRequestNum","SRType","MethodReceived","CreatedDate","SRStatus","StatusDate","Agency","Neighborhood"])
    for i in range(8000):
        n=random.choices(N,[x+random.random()*4 for x in W])[0]; d=when()
        st=random.choices(["Closed","Open","New"],[70,20,10])[0]
        w.writerow([i,random.choices(SR,[8,10,6,7,5,6,4,2,2])[0],"Phone",d.strftime("%m/%d/%Y %I:%M:%S %p"),st,"","DPW",n])
