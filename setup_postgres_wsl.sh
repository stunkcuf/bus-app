#!/bin/bash

# Script to set up PostgreSQL in WSL Ubuntu

echo "Setting up PostgreSQL in WSL..."

# Update package list
sudo apt update

# Install PostgreSQL
echo "Installing PostgreSQL..."
sudo apt install -y postgresql postgresql-contrib

# Start PostgreSQL service
sudo service postgresql start

# Set password for postgres user
echo "Setting up postgres user..."
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'localpass123';"

# Create the fleet_management database
echo "Creating fleet_management database..."
sudo -u postgres createdb fleet_management

# Convert the backup file to Unix format (remove Windows line endings)
echo "Converting backup file..."
dos2unix /mnt/c/Users/mycha/hs-bus/utilities/railway_backup.sql 2>/dev/null || true

# Import the data
echo "Importing data..."
sudo -u postgres psql -d fleet_management -f /mnt/c/Users/mycha/hs-bus/utilities/railway_backup.sql

# Configure PostgreSQL to accept connections from Windows
echo "Configuring PostgreSQL for Windows access..."
PG_VERSION=$(ls /etc/postgresql/)
sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" /etc/postgresql/$PG_VERSION/main/postgresql.conf
echo "host    all             all             172.0.0.0/8            md5" | sudo tee -a /etc/postgresql/$PG_VERSION/main/pg_hba.conf

# Restart PostgreSQL
sudo service postgresql restart

# Get WSL IP address
WSL_IP=$(ip addr show eth0 | grep -oP '(?<=inet\s)\d+(\.\d+){3}')

echo ""
echo "PostgreSQL setup complete!"
echo ""
echo "Connection details:"
echo "==================="
echo "From WSL: postgresql://postgres:localpass123@localhost:5432/fleet_management"
echo "From Windows: postgresql://postgres:localpass123@$WSL_IP:5432/fleet_management"
echo ""
echo "To start PostgreSQL in WSL: sudo service postgresql start"
echo ""
echo "Test the connection with:"
echo "psql -U postgres -d fleet_management -c 'SELECT COUNT(*) FROM buses;'"