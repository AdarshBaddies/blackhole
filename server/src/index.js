const express = require('express');
const cors = require('cors');
const sharp = require('sharp');
const fs = require('fs');
const path = require('path');
require('dotenv').config();

const app = express();
app.use(cors());
app.use(express.json());
app.use('/data', express.static(path.join(__dirname, '../data')));

const UPLOADS_DIR = path.join(__dirname, '../data/uploads');
const TEXTURES_DIR = path.join(__dirname, '../data/textures');
const METADATA_PATH = path.join(__dirname, '../data/metadata.json');

// Ensure directories exist
[UPLOADS_DIR, TEXTURES_DIR].forEach(dir => {
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
});

if (!fs.existsSync(METADATA_PATH)) {
    fs.writeFileSync(METADATA_PATH, JSON.stringify({ chunks: [], images: [] }));
}

app.post('/api/upload', async (req, res) => {
    // Placeholder for payment verification
    // In a real app, we'd check Stripe/Crypto status here

    const { imageData, x, y, z } = req.body;
    if (!imageData) return res.status(400).send('No image data');

    try {
        const buffer = Buffer.from(imageData.split(',')[1], 'base64');
        const filename = `img_${Date.now()}.png`;
        const filepath = path.join(UPLOADS_DIR, filename);

        await sharp(buffer).resize(512, 512).toFile(filepath);

        // Update metadata
        const metadata = JSON.parse(fs.readFileSync(METADATA_PATH));

        // Map x/y to Lat/Lon for the globe
        const lat = (y / 500) * 80; // Clamp to +/- 80
        const lon = (x / 500) * 160; // Clamp to +/- 160

        metadata.images.push({
            filename,
            pos: { x, y, z }, // Keep original for reference
            geo: { lat, lon }
        });

        // Image Stitching Logic: 
        // If we reach 4 images, merge them into a 1024x1024 texture chunk
        if (metadata.images.length % 4 === 0) {
            await createMegaTexture(metadata);
        }

        fs.writeFileSync(METADATA_PATH, JSON.stringify(metadata, null, 2));
        res.send({ success: true });
    } catch (err) {
        console.error(err);
        res.status(500).send('Processing error');
    }
});

async function createMegaTexture(metadata) {
    const last4 = metadata.images.slice(-4);
    const canvas = sharp({
        create: {
            width: 1024,
            height: 1024,
            channels: 4,
            background: { r: 0, g: 0, b: 0, alpha: 0 }
        }
    });

    const composites = last4.map((img, i) => ({
        input: path.join(UPLOADS_DIR, img.filename),
        top: Math.floor(i / 2) * 512,
        left: (i % 2) * 512
    }));

    const chunkFilename = `chunk_${Date.now()}.png`;
    await canvas.composite(composites).toFile(path.join(TEXTURES_DIR, chunkFilename));

    metadata.chunks.push({
        filename: chunkFilename,
        images: last4.map(img => img.filename)
    });
}

app.get('/api/gallery', (req, res) => {
    const metadata = JSON.parse(fs.readFileSync(METADATA_PATH));
    res.json(metadata);
});

const PORT = process.env.PORT || 3001;
app.listen(PORT, () => console.log(`Server running on port ${PORT}`));
