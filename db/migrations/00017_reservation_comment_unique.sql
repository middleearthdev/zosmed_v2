-- +goose Up
-- Idempotensi reserve (ADR-007 §3.3, isu #6b): satu reservasi per (akun, komentar).
-- Re-run comment:ingest (asynq retry) tidak boleh membuat reservasi/decrement
-- stok ganda. CreateReservation (db/query/reservations.sql) memakai
-- ON CONFLICT (account_id, ig_comment_id) DO NOTHING RETURNING * di atas
-- constraint ini; caller (libs/kits/seller ReservationService.Reserve) menangani
-- konflik dengan mengambil reservasi yang sudah ada via GetReservationByComment.
-- Catatan: data lama duplikat (jika ada) harus dibersihkan sebelum migrasi ini
-- berjalan di lingkungan dengan data pre-existing.
ALTER TABLE reservation
    ADD CONSTRAINT reservation_account_comment_uq UNIQUE (account_id, ig_comment_id);

-- +goose Down
ALTER TABLE reservation
    DROP CONSTRAINT reservation_account_comment_uq;
