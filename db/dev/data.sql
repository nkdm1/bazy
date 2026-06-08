INSERT INTO address (id, postcode, city, street, flat_number, number) VALUES
(105832, '03-901', 'Warszawa', 'al. Księcia Józefa Poniatowskiego', NULL, '1'),
(294711, '54-118', 'Wrocław', 'al. Śląska', NULL, '1'),
(381920, '09-400', 'Płock', 'ul. Ignacego Łukasiewicza', '12', '34'),
(472839, '02-528', 'Warszawa', 'ul. Rakowiecka', '5', '2A');

INSERT INTO users (id, username, password, name, surname, email) VALUES
(509211, 'tkwiatkowski', '$2b$12$eImiTXuWVxfM37uY4JANjQ==', 'Tomasz', 'Kwiatkowski', 't.kwiatkowski@pzpn.pl'),
(618392, 'dstefanski', '$2b$12$Kjh398dsa98321mds0921M==', 'Daniel', 'Stefański', 'd.stefanski@pzpn.pl'),
(729403, 'admin_obsada', '$2b$12$ZmxqOWevxc8921klzxczxc==', 'Adam', 'Wiśniewski', 'obsada@pzpn.pl');

INSERT INTO teams (id, name, city) VALUES
(810293, 'Legia Warszawa', 'Warszawa'),
(821304, 'Śląsk Wrocław', 'Wrocław'),
(832415, 'Lech Poznań', 'Poznań'),
(843526, 'Jagiellonia Białystok', 'Białystok');

INSERT INTO venues (id, gym_name, address_id) VALUES
(914253, 'PGE Narodowy', 105832),
(925364, 'Tarczyński Arena Wrocław', 294711);

INSERT INTO matches_level (id, match_level) VALUES
(110293, 'PKO BP Ekstraklasa'),
(121304, 'Fortuna 1 Liga');

INSERT INTO role_in_match (id, role_name) VALUES
(210984, 'Sędzia Główny'),
(221095, 'Sędzia Asystent nr 1'),
(232106, 'Sędzia Techniczny');

INSERT INTO license_names (id, name) VALUES
(319283, 'Licencja PZPN Ekstraklasa / VAR'),
(320394, 'Licencja PZPN Szczebel Centralny');

INSERT INTO referees (id, user_id, address_id, phone) VALUES
(418273, 509211, 381920, '+48600111222'),
(429384, 618392, 472839, '+48500333444');

INSERT INTO mail_verification (id, user_id, email, verification_code, expire_time, status) VALUES
(517263, 509211, 't.kwiatkowski@pzpn.pl', '258491', '2026-06-01 12:00:00', 'used'),
(528374, 618392, 'd.stefanski@pzpn.pl', '901283', '2026-06-05 23:59:59', 'pending');

INSERT INTO set_password (id, user_id, reset_token, expire_time, status) VALUES
(616253, 509211, 'f47ac10b-58cc-4372-a567-0e02b2c3d479', '2026-05-15 14:00:00', 'used'),
(627364, 618392, '550e8400-e29b-41d4-a716-446655440000', '2026-06-02 20:00:00', 'pending');

INSERT INTO wages (id, match_level_id, fee) VALUES
(715243, 110293, 4200.00),
(726354, 121304, 2100.00);

INSERT INTO matches (id, match_date, match_level_id, venue_id, home_team_id, away_team_id, status) VALUES
(814233, '2026-05-25 18:00:00', 110293, 925364, 821304, 810293, 'Finished'),
(825344, '2026-06-15 20:30:00', 110293, 914253, 810293, 832415, 'Scheduled');

INSERT INTO licence (id, referee_id, license_name_id, licence_number, issue_date, expire_date) VALUES
(913223, 418273, 319283, 'KS/PZPN/2026/015', '2026-01-01', '2026-12-31'),
(924334, 429384, 319283, 'KS/PZPN/2026/028', '2026-01-01', '2026-12-31');

INSERT INTO availability (id, referee_id, date_from, date_to) VALUES
(102312, 418273, '2026-06-01 00:00:00', '2026-06-07 23:59:59'),
(113423, 429384, '2026-06-20 08:00:00', '2026-06-25 20:00:00');

INSERT INTO match_assignments (id, referee_id, match_id, role_id, assignment_status) VALUES
(201201, 418273, 825344, 210984, 'Confirmed'),
(212312, 429384, 825344, 221095, 'Awaiting confirmation');

INSERT INTO payouts (id, referee_id, match_id, wages_id, amount, paid_at) VALUES
(309190, 418273, 814233, 715243, 4200.00, '2026-05-28 12:45:00'),
(310201, 429384, 814233, 715243, 2300.00, '2026-05-28 12:45:00');

INSERT INTO reviews (id, referee_id, match_id, rating, created_by, created_at) VALUES
(408189, 418273, 814233, 8.4, 729403, '2026-05-26 10:15:00'),
(419290, 429384, 814233, 7.9, 729403, '2026-05-26 10:30:00');