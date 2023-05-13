CREATE TABLE product (
    id INT NOT NULL AUTO_INCREMENT,
    details VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    price DOUBLE NOT NULL,
    category_id INT NOT NULL,
    PRIMARY KEY (id)
);