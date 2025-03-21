CREATE TABLE `qa_records` (
     	`id` int NOT NULL AUTO_INCREMENT,
     	`timestamp` datetime NOT NULL,
     	`responsetime` int NOT NULL,
     	`modelname` varchar(255) NOT NULL,
     	`maxtokens` int NOT NULL,
     	`temperature` float NOT NULL,
     	`system` varchar(2048) NOT NULL,
     	`question` text NOT NULL,
	`answer` text NOT NULL,
      	`itokens` int NOT NULL,
      	`otokens` int NOT NULL,
      	`stopreason` varchar(256) NOT NULL,
	PRIMARY KEY (`id`),
	FULLTEXT INDEX ft_question_answer(question, answer)
) ENGINE=Mroonga DEFAULT CHARSET=utf8mb4;