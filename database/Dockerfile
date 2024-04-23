FROM mysql:latest

# Add the script to the image
COPY bank_init3.sh /usr/local/bin/bank_init3.sh

# Make the script executable
RUN chmod +x /usr/local/bin/bank_init3.sh




EXPOSE 3306
# ENV MYSQL_ROOT_PASSWORD=root
# ENV MYSQL_DATABASE=upi