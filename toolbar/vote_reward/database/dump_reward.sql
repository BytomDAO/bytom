# ************************************************************
# Sequel Pro SQL dump
# Version 4541
#
# http://www.sequelpro.com/
# https://github.com/sequelpro/sequelpro
#
# Host: 127.0.0.1 (MySQL 5.7.24)
# Database: vote_reward
# Generation Time: 2019-07-22 13:41:50 +0000
# ************************************************************


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


# Dump of table chain_statuses
# ------------------------------------------------------------

DROP TABLE IF EXISTS `chain_statuses`;

CREATE TABLE `chain_statuses` (
  `block_height` int(11) NOT NULL,
  `block_hash` varchar(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;



# Dump of table utxos
# ------------------------------------------------------------

DROP TABLE IF EXISTS `utxos`;

CREATE TABLE `utxos` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `output_id` varchar(64) NOT NULL,
  `xpub` varchar(128) NOT NULL,
  `vote_address` varchar(62) NOT NULL,
  `vote_num` bigint(21) NOT NULL,
  `vote_height` int(11) NOT NULL,
  `veto_height` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `output_id` (`output_id`),
  KEY `xpub` (`xpub`),
  KEY `vote_height` (`vote_height`),
  KEY `veto_height` (`veto_height`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;




/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
