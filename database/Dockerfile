FROM mysql:latest
# Add the script to the image
COPY bank_init1.sh /usr/local/bin/bank_init1.sh

# Make the script executable
RUN chmod +x /usr/local/bin/bank_init1.sh
EXPOSE 3306
# ENV MYSQL_ROOT_PASSWORD=root
# ENV MYSQL_DATABASE=upi