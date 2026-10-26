-- Наполнение таблиц лабораторной №3. Выполняется через Adminer (http://localhost:8081)
-- после создания таблиц командой go run ./cmd/migrate

-- Пароль у всех четырёх учебных проектировщиков один: lumicalc-demo.
-- В таблице хранится только его хэш bcrypt
INSERT INTO lighting_users (login, full_name, password_hash) VALUES
  ('a.parahnevich', 'Парахневич А.', '$2a$10$nvegcA.R9Z6bxVTJVtFsEOLIFz/18TvnwY4SvPc9AgeHosifdF0oG'),
  ('i.svetov',      'Светов И. В.',  '$2a$10$nvegcA.R9Z6bxVTJVtFsEOLIFz/18TvnwY4SvPc9AgeHosifdF0oG'),
  ('m.lampova',     'Лампова М. С.', '$2a$10$nvegcA.R9Z6bxVTJVtFsEOLIFz/18TvnwY4SvPc9AgeHosifdF0oG'),
  ('d.luxov',       'Люксов Д. А.',  '$2a$10$nvegcA.R9Z6bxVTJVtFsEOLIFz/18TvnwY4SvPc9AgeHosifdF0oG');

-- В image_name и video_name хранятся только имена файлов в бакете Minio,
-- адрес файла веб-сервис собирает при ответе.
-- Три обязательных статуса: опубликован, удалён, черновик.
-- Черновик принадлежит проектировщику №2, поэтому текущий проектировщик (№1)
-- может добавить свой светильник: черновик у каждого не более одного
INSERT INTO lighting_equipment
  (equipment_name, short_description, equipment_status, image_name, video_name, power_watt, beam_angle_deg, created_at, formed_at, creator_id) VALUES
  ('SMART+ MATTER Filament Multicolor Classic Edison 40 4.8W E27',
   'Умная филаментная лампа с поддержкой Matter и настройкой цвета, тёплый свет для жилых помещений.',
   'опубликован',
   'vintage-1906-classic-a.jpg',
   'smart-matter-classic-a.mp4',
   4.80, 300, '2026-01-12 10:00:00', '2026-01-13 09:30:00', 1),

  ('Vintage 1906 Edison 28 Filament 4W 820 Gold E27',
   'Филаментная лампа Edison с золотистой колбой, для акцентного декоративного освещения.',
   'опубликован',
   'vintage-1906-edison-28-gold-e27.jpg',
   NULL,
   4.00, 300, '2026-01-20 12:15:00', '2026-01-21 08:00:00', 1),

  ('Vintage 1906 Big Oval 10 Filament Magnetic 4W 818 Smoke E27',
   'Большая овальная филаментная лампа с дымчатой колбой, магнитный стиль нити накала.',
   'опубликован',
   'vintage-1906-big-oval-10-smoke-e27.jpg',
   NULL,
   4.00, 320, '2026-02-02 14:40:00', '2026-02-03 11:20:00', 2),

  ('Vintage 1906 Classic B 40 Filament DIM 4.8W 827 Clear E14',
   'Диммируемая филаментная свеча с прозрачной колбой, цоколь E14, для декоративных светильников.',
   'опубликован',
   'vintage-1906-classic-b40-clear-e14.jpg',
   NULL,
   4.80, 320, '2026-02-10 09:05:00', '2026-02-11 10:10:00', 2),

  ('Vintage 1906 Big Elipse 6 Filament Magnetic 4W 818 Smoke E27',
   'Компактная эллиптическая филаментная лампа с дымчатой колбой, магнитный стиль нити накала.',
   'удалён',
   'vintage-1906-big-elipse-6-smoke-e27.jpg',
   NULL,
   4.00, 320, '2026-02-15 16:30:00', NULL, 1),

  ('Vintage 1906 Edison 22 Filament 2.5W 824 Gold E27',
   NULL,
   'черновик',
   NULL, NULL,
   NULL, NULL, '2026-03-01 08:20:00', NULL, 2);

-- Лайки: связь м-м, один пользователь может лайкнуть несколько светильников
INSERT INTO lighting_equipment_likes (user_id, equipment_id) VALUES
  (1, 1), (2, 1), (3, 1), (4, 1),
  (1, 2), (3, 2),
  (2, 3), (4, 3), (3, 3),
  (1, 4), (2, 4), (3, 4), (4, 4);
