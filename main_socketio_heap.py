from flask import Flask
from flask_socketio import SocketIO, emit
import heapq
from collections import defaultdict
import threading

app = Flask(__name__)
socketio = SocketIO(app)

class Order:
    def __init__(self, user_id, asset, order_type, price, amount):
        self.user_id = user_id
        self.asset = asset
        self.order_type = order_type
        self.price = price
        self.amount = amount

    def __lt__(self, other):
        if self.order_type == 'buy':
            return self.price > other.price  
        else:
            return self.price < other.price 

class OrderBook:
    def __init__(self):
        self.buy_orders = []  
        self.sell_orders = [] 

    def add_order(self, order):
        if order.order_type == 'buy':
            heapq.heappush(self.buy_orders, order)
        else:
            heapq.heappush(self.sell_orders, order)
        self.match_orders()

    def match_orders(self):
        while self.buy_orders and self.sell_orders:
            best_buy = self.buy_orders[0]
            best_sell = self.sell_orders[0]

            if best_buy.price >= best_sell.price:
                # Match found
                matched_amount = min(best_buy.amount, best_sell.amount)
                print(f"Matching {matched_amount} of {best_buy.user_id} (buy) with {best_sell.user_id} (sell) at price {best_sell.price}")
                emit('order_matched', {'status': 'success', 
                                        'matched_amount': matched_amount, 
                                        'buyer': best_buy.user_id, 
                                        'seller': best_sell.user_id, 
                                        'price': best_sell.price}, 
                                        broadcast=True)
                # Update amounts
                best_buy.amount -= matched_amount
                best_sell.amount -= matched_amount

                # Remove orders if fully matched
                if best_buy.amount == 0:
                    heapq.heappop(self.buy_orders)
                if best_sell.amount == 0:
                    heapq.heappop(self.sell_orders)
            else:
                break  # No more matches possible

class MatchingEngine:
    def __init__(self):
        self.order_books = defaultdict(OrderBook)
        self.lock = threading.Lock()

    def add_order(self, user_id, asset, order_type, price, amount):
        order = Order(user_id, asset, order_type, price, amount)
        with self.lock:
            self.order_books[asset].add_order(order)

# Initialize the matching engine
engine = MatchingEngine()

@socketio.on('new_order')
def handle_new_order(data):
    print(data)
    user_id = data['user_id']
    asset = data['asset']
    order_type = data['order_type']
    price = data['price']
    amount = data['amount']
    
    engine.add_order(user_id, asset, order_type, price, amount)
    emit('order_received', {'status': 'success'}, broadcast=True)

if __name__ == "__main__":
    socketio.run(app, host='0.0.0.0', port=5001)