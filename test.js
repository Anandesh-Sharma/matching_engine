const io = require('socket.io-client');
const socket = io('http://localhost:5001');

socket.on('connect', () => {
    console.log('Connected to the server');
});

socket.on('order_received', (data) => {
    console.log('Order received:', data);
});


socket.on('order_matched', (data) => {
    console.log('Order matched:', data);
});


// Example of sending an order
socket.emit('new_order', {
    user_id: 'user1',
    asset: 'Asset1',
    order_type: 'buy',
    price: 10.50,
    amount: 60
});


socket.emit('new_order', {
    user_id: 'user2',
    asset: 'Asset1',
    order_type: 'sell',
    price: 10.50,
    amount: 60
});