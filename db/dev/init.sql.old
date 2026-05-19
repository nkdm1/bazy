CREATE TABLE `users` (
  `id` integer PRIMARY KEY,
  `name` varchar(100),
  `surname` varchar(100),
  `role` ENUM('admin','referee','viewer') DEFAULT 'viewer',
  `created_at` timestamp
);

CREATE TABLE `mail_verification` (
  `id` integer PRIMARY KEY,
  `user_id` integer NOT NULL,
  `mail_given` varchar(255),
  `verification_code` varchar(255),
  `send_time` timestamp,
  `expire_time` timestamp,
  `is_verifed` boolean DEFAULT false,
  `status` ENUM('not_active','active','expired','used') DEFAULT 'not_active'
);

CREATE TABLE `set_password` (
  `id` integer PRIMARY KEY,
  `user_id` integer NOT NULL,
  `reset_token` varchar(255),
  `password_hash` varchar(255),
  `created_at` timestamp,
  `expire_time` timestamp,
  `status` ENUM('active','expired','used') DEFAULT 'active'
);

CREATE TABLE `address` (
  `id` integer PRIMARY KEY,
  `postcode` varchar(6),
  `city` varchar(100),
  `street` varchar(100),
  `street_number` varchar(100),
  `flat_number` integer
);

CREATE TABLE `referees` (
  `id` integer PRIMARY KEY,
  `address_id` integer NOT NULL,
  `user_id` integer UNIQUE,
  `phone` varchar(255)
);

CREATE TABLE `licenses_names` (
  `id` integer PRIMARY KEY,
  `license_name` varchar(255) NOT NULL
);

CREATE TABLE `licenses` (
  `id` integer PRIMARY KEY,
  `referees_id` integer NOT NULL,
  `license_number` varchar(255) UNIQUE,
  `license_name` integer NOT NULL,
  `issued_at` date,
  `expire_at` date
);

CREATE TABLE `availability` (
  `id` integer PRIMARY KEY,
  `referee_id` integer NOT NULL,
  `date_from` date NOT NULL,
  `date_to` date NOT NULL
);

CREATE TABLE `teams` (
  `id` integer PRIMARY KEY,
  `name` varchar(255),
  `city` varchar(100)
);

CREATE TABLE `matches_level` (
  `id` integer PRIMARY KEY,
  `match_level` ENUM('FIBA','PLK','centralna','okregowa','stazysta')
);

CREATE TABLE `venues` (
  `id` integer PRIMARY KEY,
  `gym_name` varchar(255),
  `address_id` integer NOT NULL
);

CREATE TABLE `matches` (
  `id` integer PRIMARY KEY,
  `match_date` timestamp,
  `level_of_match` integer NOT NULL,
  `venue_id` integer,
  `home_team_id` integer,
  `away_team_id` integer,
  `status` ENUM('Scheduled','Completed','Cancelled') DEFAULT 'Scheduled',
  `home_team_points` integer,
  `away_team_points` integer
);

CREATE TABLE `role_in_match` (
  `id` integer PRIMARY KEY,
  `match_role` ENUM('Crew_Chief','Umpire') DEFAULT 'Umpire'
);

CREATE TABLE `match_assignments` (
  `id` integer PRIMARY KEY,
  `referee_id` integer NOT NULL,
  `match_id` integer NOT NULL,
  `role` integer NOT NULL,
  `assignment_status` ENUM('pending','accepted','declined') DEFAULT 'pending'
);

CREATE TABLE `wages` (
  `id` integer PRIMARY KEY,
  `match_level` integer NOT NULL,
  `role_in_match` integer NOT NULL,
  `fee` decimal(10,2) NOT NULL
);

CREATE TABLE `payouts` (
  `id` integer PRIMARY KEY,
  `referee_id` integer NOT NULL,
  `match_id` integer NOT NULL,
  `wages_id` integer NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `status` ENUM('Pending','Paid') DEFAULT 'Pending',
  `paid_at` timestamp
);

CREATE TABLE `reviews` (
  `id` integer PRIMARY KEY,
  `referee_id` integer NOT NULL,
  `rating` integer NOT NULL,
  `created_at` timestamp,
  `created_by` integer NOT NULL
);

-- --- Relationships / Constraints ---

ALTER TABLE `referees` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);
ALTER TABLE `referees` ADD FOREIGN KEY (`address_id`) REFERENCES `address` (`id`);

ALTER TABLE `licenses` ADD FOREIGN KEY (`referees_id`) REFERENCES `referees` (`id`);
ALTER TABLE `licenses` ADD FOREIGN KEY (`license_name`) REFERENCES `licenses_names` (`id`);

ALTER TABLE `availability` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);

ALTER TABLE `venues` ADD FOREIGN KEY (`address_id`) REFERENCES `address` (`id`);

ALTER TABLE `matches` ADD FOREIGN KEY (`home_team_id`) REFERENCES `teams` (`id`);
ALTER TABLE `matches` ADD FOREIGN KEY (`away_team_id`) REFERENCES `teams` (`id`);
ALTER TABLE `matches` ADD FOREIGN KEY (`venue_id`) REFERENCES `venues` (`id`);
ALTER TABLE `matches` ADD FOREIGN KEY (`level_of_match`) REFERENCES `matches_level` (`id`);

ALTER TABLE `match_assignments` ADD FOREIGN KEY (`match_id`) REFERENCES `matches` (`id`);
ALTER TABLE `match_assignments` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);
ALTER TABLE `match_assignments` ADD FOREIGN KEY (`role`) REFERENCES `role_in_match` (`id`);

ALTER TABLE `wages` ADD FOREIGN KEY (`match_level`) REFERENCES `matches_level` (`id`);
ALTER TABLE `wages` ADD FOREIGN KEY (`role_in_match`) REFERENCES `role_in_match` (`id`);

ALTER TABLE `payouts` ADD FOREIGN KEY (`match_id`) REFERENCES `matches` (`id`);
ALTER TABLE `payouts` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);
ALTER TABLE `payouts` ADD FOREIGN KEY (`wages_id`) REFERENCES `wages` (`id`);

ALTER TABLE `reviews` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);
ALTER TABLE `reviews` ADD FOREIGN KEY (`created_by`) REFERENCES `users` (`id`);

ALTER TABLE `mail_verification` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);
ALTER TABLE `set_password` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);
