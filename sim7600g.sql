CREATE TABLE `gps` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `hostname` varchar(255) NOT NULL,
  `record_timestamp` timestamp NULL DEFAULT current_timestamp(),
  `gps_timestamp` timestamp NOT NULL,
  `class` varchar(10) NOT NULL,
  `tag` varchar(255) NOT NULL,
  `device` varchar(255) NOT NULL,
  `mode` int(11) NOT NULL,
  `ept` float NOT NULL,
  `lat` float NOT NULL,
  `lon` float NOT NULL,
  `alt` float NOT NULL,
  `epx` float NOT NULL,
  `epy` float NOT NULL,
  `epv` float NOT NULL,
  `track` float NOT NULL,
  `speed` float NOT NULL,
  `climb` float NOT NULL,
  `epd` float NOT NULL,
  `eps` float NOT NULL,
  `epc` float NOT NULL,
  PRIMARY KEY (`id`),
  KEY `gps_record_timestamp_IDX` (`record_timestamp`) USING BTREE,
  KEY `gps_gps_timestamp_IDX` (`gps_timestamp`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=591660 DEFAULT CHARSET=utf8