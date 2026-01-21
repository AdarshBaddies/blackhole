import * as THREE from 'three';
import { WebGPURenderer, NodeFrame, MeshBasicNodeMaterial, MeshStandardNodeMaterial } from 'three/webgpu';
import { texture, uv, vec2, vec3, color, mix, floor, distance, cameraPosition, positionWorld, time, mx_noise_float, attribute, rotate, length, uniform } from 'three/tsl';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

let renderer, scene, camera, controls;
const nodeFrame = new NodeFrame();
const GLOBE_RADIUS = 50;

let instancedChunks = [];

async function init() {
    console.log('Initializing Humanity Globe...');


    //
    const audio = document.getElementById('bg-audio');
    const audioBtn = document.getElementById('audio-toggle');
    let hasStarted = false;

    // A. Start music on the first click anywhere on the page
    window.addEventListener('click', () => {
        if (!hasStarted) {
            audio.play();
            audioBtn.innerText = "🔊";
            hasStarted = true;
        }
    }, { once: true }); // {once: true} makes this listener run only once

    // B. Toggle Pause/Play with the button
    audioBtn.addEventListener('click', (e) => {
        e.stopPropagation(); // Prevents the 'window click' from triggering
        if (audio.paused) {
            audio.play();
            audioBtn.innerText = "🔊";
        } else {
            audio.pause();
            audioBtn.innerText = "🔇";
        }
    });

    //

    const addBtn = document.getElementById('add-btn');
    if (addBtn) {
        addBtn.addEventListener('click', testUpload);
    }

    try {
        renderer = new WebGPURenderer({ antialias: true });
        await renderer.init();

        renderer.setPixelRatio(window.devicePixelRatio);
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.setAnimationLoop(animate);
        document.body.appendChild(renderer.domElement);

        scene = new THREE.Scene();
        scene.background = new THREE.Color(0x00000a);

        camera = new THREE.PerspectiveCamera(60, window.innerWidth / window.innerHeight, 0.1, 2000);
        camera.position.set(0, 0, 150);

        controls = new OrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        controls.minDistance = 51; // Just outside the globe
        controls.maxDistance = 500;

        // --- LIGHTING ---
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.8);
        scene.add(ambientLight);

        const sunLight = new THREE.DirectionalLight(0xffffff, 2);
        sunLight.position.set(100, 100, 100);
        scene.add(sunLight);

        // --- HUMANITY GLOBE ---
        const globeGeo = new THREE.SphereGeometry(GLOBE_RADIUS, 64, 64);
        const globeMat = new THREE.MeshStandardMaterial({
            color: 0x112244,
            transparent: true,
            opacity: 0.8,
            wireframe: true
        });
        const globe = new THREE.Mesh(globeGeo, globeMat);
        scene.add(globe);

        // --- LOAD INITIAL GALLERY ---
        await refreshGallery();

        window.addEventListener('resize', onWindowResize);
    } catch (err) {
        console.error('Three.js Initialization failed:', err);
    }
}

async function refreshGallery() {
    console.log('Refreshing globe data...');
    try {
        const res = await fetch('http://localhost:3001/api/gallery');
        const data = await res.json();
        console.log('Gallery data received:', data);

        instancedChunks.forEach(c => scene.remove(c.mesh));
        instancedChunks = [];

        const loader = new THREE.TextureLoader();

        for (const chunkData of data.chunks) {
            const url = `http://localhost:3001/data/textures/${chunkData.filename}`;
            try {
                const textureMap = await loader.loadAsync(url);
                createChunkMesh(textureMap, chunkData);
            } catch (err) {
                console.error('Failed to load chunk texture:', url, err);
            }
        }

        const unstitched = data.images.slice(data.chunks.length * 4);
        unstitched.forEach(img => {
            const url = `http://localhost:3001/data/uploads/${img.filename}`;
            loader.load(url, (tex) => {
                createSingleImageMesh(tex, img.pos);
            });
        });

    } catch (err) {
        console.error('Failed to refresh gallery:', err);
    }
}

function sphericalToCartesian(lat, lon, radius) {
    const phi = (90 - lat) * (Math.PI / 180);
    const theta = (lon + 180) * (Math.PI / 180);

    return new THREE.Vector3(
        -radius * Math.sin(phi) * Math.cos(theta),
        radius * Math.cos(phi),
        radius * Math.sin(phi) * Math.sin(theta)
    );
}

function createChunkMesh(tex, data) {
    // For now, mapping chunks to random spherical points
    const lat = (Math.random() - 0.5) * 160;
    const lon = (Math.random() - 0.5) * 320;
    const pos = sphericalToCartesian(lat, lon, GLOBE_RADIUS + 0.5);

    const geometry = new THREE.PlaneGeometry(10, 10);
    const material = new MeshStandardNodeMaterial();

    // Zoom-to-Flat is handled by positioning logic + pixelation shader
    const dist = distance(positionWorld, cameraPosition);
    const pixelSize = mix(5.0, 1024.0, dist.sub(55.0).div(20.0).clamp(0, 1)); // Sharpen as we get close
    const pUV = floor(uv().mul(pixelSize)).div(pixelSize);
    material.colorNode = texture(tex, pUV);

    const mesh = new THREE.Mesh(geometry, material);
    mesh.position.copy(pos);
    mesh.lookAt(0, 0, 0);
    mesh.renderOrder = 1;

    scene.add(mesh);
    instancedChunks.push({ mesh });
}

function createSingleImageMesh(tex, posData) {
    // If backend still sends x, y, z, wrap them into lat/lon
    const lat = (posData.y / 500) * 80;
    const lon = (posData.x / 500) * 160;
    const pos = sphericalToCartesian(lat, lon, GLOBE_RADIUS + 0.5);

    const geometry = new THREE.PlaneGeometry(5, 5);
    const material = new MeshStandardNodeMaterial();
    material.colorNode = texture(tex);

    const mesh = new THREE.Mesh(geometry, material);
    mesh.position.copy(pos);
    mesh.lookAt(0, 0, 0);
    mesh.renderOrder = 2;

    scene.add(mesh);
}

async function testUpload() {
    const fileInput = document.getElementById('file-input');
    fileInput.click(); // Trigger file picker

    fileInput.onchange = async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        const reader = new FileReader();
        reader.onload = async (event) => {
            const imageData = event.target.result;

            try {
                // Generate random lat/lon for the upload
                const x = (Math.random() - 0.5) * 1000;
                const y = (Math.random() - 0.5) * 1000;

                const res = await fetch('http://localhost:3001/api/upload', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        imageData,
                        x, y, z: 0
                    })
                });
                const data = await res.json();
                if (data.success) {
                    alert('Success! Your image has been pasted onto the globe.');
                    refreshGallery();
                }
            } catch (err) {
                console.error('Upload failed:', err);
                alert('Upload failed. Check if the server is running on port 3001.');
            }
        };
        reader.readAsDataURL(file);
    };
}

function onWindowResize() {
    camera.aspect = window.innerWidth / window.innerHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(window.innerWidth, window.innerHeight);
}

function animate() {
    nodeFrame.update();
    if (controls) controls.update();
    renderer.render(scene, camera);
}

function showError(msg) {
    const ui = document.getElementById('ui');
    if (ui) {
        const errorMsg = document.createElement('p');
        errorMsg.style.color = '#ff4444';
        errorMsg.style.background = 'rgba(0,0,0,0.5)';
        errorMsg.style.padding = '10px';
        errorMsg.innerText = msg;
        ui.appendChild(errorMsg);
    }
}

init();
