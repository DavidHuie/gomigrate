CREATE TABLE test (
     id INT NOT NULL AUTO_INCREMENT,
     PRIMARY KEY (id)
);

SELECT 'my comment' AS comment;

CREATE TABLE test2 (
     id INT NOT NULL AUTO_INCREMENT,
     PRIMARY KEY (id)
);

SELECT 'my other comment' AS comment;

CREATE TABLE tt (c text NOT NULL);

INSERT INTO tt VALUES('a');
INSERT INTO tt VALUES('x');
