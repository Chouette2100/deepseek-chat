CREATE TABLE `misc`.`user` (
  `email` VARCHAR(128) NOT NULL,
  `pswd` VARCHAR(128) NOT NULL,
  `ts` DATETIME NOT NULL,
  PRIMARY KEY (`email`));
