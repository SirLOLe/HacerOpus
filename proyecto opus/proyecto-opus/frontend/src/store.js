import { writable } from 'svelte/store';

// Estos son los valores por defecto al abrir la aplicación
export const appSettings = writable({
	bitrate: "128k",
	coverSize: 700,
	quality: 2,
	replaceCover: false,
	compressionLevel: 10, // 10 es la máxima eficiencia del algoritmo Opus
	threads: 4            // Hilos concurrentes
});