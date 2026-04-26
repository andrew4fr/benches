let myMap;
let userMarker;
let benchMarkers = [];
let userPosition = null;
let selectedCoords = null;
let selectedPhoto = null;

// Скамейки будут загружены с бэкэнда
let benches = [];

const API_BASE_URL = 'http://localhost:8080/api';

ymaps.ready(init);

function init() {
    // Определение случайного центра карты (северная Америка)
    const randomCenters = [
        [40.7128, -74.0060],  // Нью-Йорк
        [34.0522, -118.2437], // Лос-Анджелес
        [51.5074, -0.1278],   // Лондон
        [48.8566, 2.3522],    // Париж
        [-33.8688, 151.2093], // Сидней
        [35.6762, 139.6503]   // Токио
    ];
    const randomCenter = randomCenters[Math.floor(Math.random() * randomCenters.length)];

    // Создание карты со случайным центром (будет изменён после геолокации)
    myMap = new ymaps.Map("map", {
        center: randomCenter,
        zoom: 10,
        controls: ['zoomControl']
    });

    // Кнопка "Моё местоположение"
    const geolocationButton = new ymaps.control.Button({
        data: {
            content: '📍',
            title: 'Моё местоположение'
        },
        options: {
            size: 'small',
            float: 'left',
            floatIndex: 1
        }
    });

    geolocationButton.events.add('click', function() {
        if (userPosition) {
            myMap.setCenter(userPosition, 15);
        } else {
            locateUser().then(() => {
                myMap.setCenter(userPosition, 15);
            });
        }
    });

    myMap.controls.add(geolocationButton);

    // Сначала загружаем скамейки, затем определяем локацию
    fetchBenches().then(() => {
        // Автоматическое определение геопозиции после загрузки скамеек
        return locateUser();
    }).then(() => {
        // Перемещаем карту к пользователю
        myMap.setCenter(userPosition, 15);

        // Обновляем список скамеек с расстояниями
        updateBenchesWithDistance();

        document.getElementById('infoPanel').classList.remove('hidden');
    }).catch((error) => {
        console.error('Ошибка:', error);
        // При ошибке геолокации показать все скамейки без расстояний
        renderBenches(benches);
    });

    // Обработка клика по карте для выбора места
    myMap.events.add('click', function (e) {
        const coords = e.get('coords');
        selectedCoords = coords;
        document.getElementById('benchCoords').value = `${coords[0].toFixed(6)}, ${coords[1].toFixed(6)}`;

        // Показываем метку выбранной точки
        if (window.selectedMarker) {
            myMap.geoObjects.remove(window.selectedMarker);
        }

        window.selectedMarker = new ymaps.Placemark(coords, {
            hintContent: 'Выбрано место'
        }, {
            preset: 'islands#blueCircleIcon'
        });

        myMap.geoObjects.add(window.selectedMarker);
    });

    // Кнопка "Найти мои скамейки"
    // Удалена - геолокация определяется автоматически

    // Кнопка "Добавить скамейку"
    document.getElementById('addBenchBtn').addEventListener('click', function() {
        document.getElementById('addBenchModal').classList.remove('hidden');
    });

    // Закрытие модального окна
    document.querySelector('.close').addEventListener('click', function() {
        document.getElementById('addBenchModal').classList.add('hidden');
    });

    window.addEventListener('click', function(e) {
        if (e.target === document.getElementById('addBenchModal')) {
            document.getElementById('addBenchModal').classList.add('hidden');
        }
    });

    // Обработка формы добавления скамейки
    document.getElementById('addBenchForm').addEventListener('submit', addBench);

    // Предпросмотр фото
    document.getElementById('benchPhoto').addEventListener('change', function(e) {
        const file = e.target.files[0];
        if (file) {
            const reader = new FileReader();
            reader.onload = function(e) {
                selectedPhoto = e.target.result;
                const preview = document.getElementById('photoPreview');
                preview.innerHTML = `<img src="${selectedPhoto}" alt="Предпросмотр">`;
                preview.style.display = 'block';
            };
            reader.readAsDataURL(file);
        } else {
            selectedPhoto = null;
            document.getElementById('photoPreview').style.display = 'none';
        }
    });
}

async function fetchBenches() {
    try {
        const response = await fetch(`${API_BASE_URL}/benches`);
        if (!response.ok) throw new Error('Ошибка загрузки');
        benches = await response.json();
        renderBenches(benches);
        return benches;
    } catch (error) {
        console.error('Не удалось загрузить скамейки:', error);
        benches = [];
        renderBenches([]);
        return benches;
    }
}

function updateBenchesWithDistance() {
    if (!userPosition || !benches) return;

    const benchesWithDistance = benches.map(bench => {
        const distance = calculateDistance(
            userPosition[0], userPosition[1],
            bench.latitude, bench.longitude
        );
        return { ...bench, distance, coords: [bench.latitude, bench.longitude] };
    });

    renderBenches(benchesWithDistance);
}

function locateUser() {
    return ymaps.geolocation.get({
        provider: 'browser',
        mapStateAutoApply: true
    }).then(function (result) {
        userPosition = result.geoObjects.get(0).geometry.getCoordinates();

        // Создание метки пользователя
        if (userMarker) {
            myMap.geoObjects.remove(userMarker);
        }

        userMarker = new ymaps.Placemark(userPosition, {
            hintContent: 'Вы здесь'
        }, {
            preset: 'islands#redDotIcon'
        });

        myMap.geoObjects.add(userMarker);

        // Фильтрация и отображение скамеек рядом
        updateBenchesWithDistance();

        document.getElementById('infoPanel').classList.remove('hidden');
    });
}

function getNearbyBenches(position, radiusMeters) {
    return benches.filter(bench => {
        const distance = calculateDistance(
            position[0], position[1],
            bench.latitude, bench.longitude
        );
        return distance <= radiusMeters;
    }).map(bench => {
        const distance = calculateDistance(
            position[0], position[1],
            bench.latitude, bench.longitude
        );
        return { ...bench, distance, coords: [bench.latitude, bench.longitude] };
    }).sort((a, b) => a.distance - b.distance);
}

function calculateDistance(lat1, lon1, lat2, lon2) {
    const R = 6371e3; // Радиус Земли в метрах
    const φ1 = lat1 * Math.PI / 180;
    const φ2 = lat2 * Math.PI / 180;
    const Δφ = (lat2 - lat1) * Math.PI / 180;
    const Δλ = (lon2 - lon1) * Math.PI / 180;

    const a = Math.sin(Δφ/2) * Math.sin(Δφ/2) +
              Math.cos(φ1) * Math.cos(φ2) *
              Math.sin(Δλ/2) * Math.sin(Δλ/2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a));

    return R * c;
}

function renderBenches(benchList) {
    // Защита от null/undefined
    if (!benchList || !Array.isArray(benchList)) {
        benchList = [];
    }

    const container = document.getElementById('benchList');

    // Очистка старых меток скамеек
    benchMarkers.forEach(marker => {
        myMap.geoObjects.remove(marker);
    });
    benchMarkers = [];

    if (benchList.length === 0) {
        container.innerHTML = '<p>Рядом с вами пока нет добавленных скамеек. Будьте первым, кто добавит!</p>';
        return;
    }

    let html = '';
    benchList.forEach((bench, index) => {
        let distanceText = '—';
        if (bench.distance !== null && bench.distance !== undefined) {
            distanceText = bench.distance < 1000
                ? `${Math.round(bench.distance)} м`
                : `${(bench.distance / 1000).toFixed(1)} км`;
        }

        const photoHtml = bench.photo ? `<img src="${bench.photo}" alt="${bench.name}" style="max-width: 100%; border-radius: 6px; margin-top: 8px;">` : '';

        const coords = bench.coords || [bench.latitude, bench.longitude];

        html += `
            <div class="bench-item" onclick="zoomToBench(${coords[0]}, ${coords[1]})" style="cursor: pointer;">
                <h3>${bench.name}</h3>
                <p>${bench.comment || 'Без описания'}</p>
                ${photoHtml}
                <span class="distance">📏 ${distanceText}</span>
            </div>
        `;

        // Добавление метки на карту
        const balloonContent = `
            <div style="max-width: 200px;">
                <strong>${bench.name}</strong><br>
                ${bench.comment || ''}<br>
                ${bench.photo ? `<img src="${bench.photo}" style="max-width: 150px; margin-top: 8px; border-radius: 4px;">` : ''}<br>
                <small>Расстояние: ${distanceText}</small>
            </div>
        `;

        const marker = new ymaps.Placemark(coords, {
            hintContent: bench.name,
            balloonContent: balloonContent
        }, {
            preset: 'islands#greenCircleDotIcon'
        });

        myMap.geoObjects.add(marker);
        benchMarkers.push(marker);
    });

    container.innerHTML = html;
}

function zoomToBench(lat, lon) {
    myMap.setCenter([lat, lon], 16);
}

function addBench(e) {
    e.preventDefault();

    if (!selectedCoords) {
        alert('Пожалуйста, выберите место на карте');
        return;
    }

    const name = document.getElementById('benchName').value;
    const comment = document.getElementById('benchComment').value;
    const email = document.getElementById('benchEmail').value;

    // Отправка данных на бэкэнд
    const benchData = {
        name: name,
        comment: comment,
        email: email,
        latitude: selectedCoords[0],
        longitude: selectedCoords[1]
    };

    if (selectedPhoto) {
        benchData.photo = selectedPhoto;
    }

    fetch(`${API_BASE_URL}/benches`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(benchData)
    })
        .then(response => {
            if (!response.ok) throw new Error('Ошибка сохранения');
            return response.json();
        })
        .then(savedBench => {
            // Добавляем новую скамейку в список
            if (benches) {
                benches.push(savedBench);
            }

            // Если есть пользовательская позиция, рассчитываем расстояние
            let distanceText = '—';
            if (userPosition) {
                const distance = calculateDistance(
                    userPosition[0], userPosition[1],
                    selectedCoords[0], selectedCoords[1]
                );
                distanceText = distance < 1000
                    ? `${Math.round(distance)} м`
                    : `${(distance / 1000).toFixed(1)} км`;
            }

            // Добавление метки на карту
            const balloonContent = `
            <div style="max-width: 200px;">
                <strong>${name}</strong><br>
                ${comment || ''}<br>
                ${selectedPhoto ? `<img src="${selectedPhoto}" style="max-width: 150px; margin-top: 8px; border-radius: 4px;">` : ''}<br>
                <small>Расстояние: ${distanceText}</small>
            </div>
        `;

            const marker = new ymaps.Placemark(selectedCoords, {
                hintContent: name,
                balloonContent: balloonContent
            }, {
                preset: 'islands#greenCircleDotIcon'
            });

            myMap.geoObjects.add(marker);
            benchMarkers.push(marker);

            // Очистка формы и закрытие модального окна
            document.getElementById('addBenchForm').reset();
            document.getElementById('addBenchModal').classList.add('hidden');
            document.getElementById('photoPreview').style.display = 'none';

            if (window.selectedMarker) {
                myMap.geoObjects.remove(window.selectedMarker);
                window.selectedMarker = null;
            }

            selectedCoords = null;
            selectedPhoto = null;

            // Обновление отображения
            if (userPosition) {
                updateBenchesWithDistance();
            } else {
                renderBenches(benches);
            }

            alert('Скамейка успешно добавлена!');
        })
        .catch(error => {
            console.error('Ошибка:', error);
            alert('Не удалось добавить скамейку. Попробуйте позже.');
        });
}
