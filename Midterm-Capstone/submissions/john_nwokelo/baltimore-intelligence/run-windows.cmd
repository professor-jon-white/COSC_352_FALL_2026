@echo off
setlocal
cd /d "%~dp0"

if not exist "data\requests.geojson" (
    echo Missing data\requests.geojson
    echo Copy your 311 download from the old project's data folder.
    pause
    exit /b 1
)
if not exist "data\neighborhoods.geojson" (
    echo Missing data\neighborhoods.geojson
    echo Copy your neighborhood download from the old project's data folder.
    pause
    exit /b 1
)

echo Running the Baltimore report with your downloaded city datasets...
echo.
go run .\cmd\engine -requests "data\requests.geojson" -boundaries "data\neighborhoods.geojson" %*
if errorlevel 1 echo ERROR: The application did not finish successfully.
echo.
pause
