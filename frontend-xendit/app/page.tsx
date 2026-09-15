"use client";

import { useState } from "react";

export default function CheckoutPage() {
  const [amount, setAmount] = useState<number | "">("");
  const [method, setMethod] = useState("BCA");
  const [email, setEmail] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleCheckout = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setErrorMessage("");

    try {
      // Pastikan backend Gin berjalan di localhost:8080
      const res = await fetch("http://localhost:8080/api/checkout", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          amount: Number(amount),
          payment_method: method,
          customer_email: email,
        }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error || "Gagal melakukan checkout");
      }

      if (data.invoice_url) {
        // Redirect otomatis ke halaman pembayaran Xendit
        window.location.href = data.invoice_url;
      } else {
        throw new Error("URL Invoice tidak ditemukan dari response server.");
      }
    } catch (error: any) {
      console.error("Checkout Error:", error);
      setErrorMessage(error.message || "Terjadi kesalahan koneksi ke server.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Selesaikan Pembayaran
        </h2>
        <p className="mt-2 text-center text-sm text-gray-600">
          Simulasi integrasi Next.js, Golang, dan Xendit
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10 border border-gray-100">
          <form className="space-y-6" onSubmit={handleCheckout}>
            {/* Input Email */}
            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium text-gray-700"
              >
                Email Pembeli
              </label>
              <div className="mt-1">
                <input
                  id="email"
                  name="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="farras@example.com"
                  className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                />
              </div>
            </div>

            {/* Input Nominal */}
            <div>
              <label
                htmlFor="amount"
                className="block text-sm font-medium text-gray-700"
              >
                Nominal Pembayaran (Rp)
              </label>
              <div className="mt-1 relative rounded-md shadow-sm">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <span className="text-gray-500 sm:text-sm">Rp</span>
                </div>
                <input
                  id="amount"
                  name="amount"
                  type="number"
                  required
                  min="10000"
                  value={amount}
                  onChange={(e) => setAmount(Number(e.target.value))}
                  placeholder="100000"
                  className="appearance-none block w-full pl-10 px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                />
              </div>
              <p className="mt-1 text-xs text-gray-500">
                Minimal pembayaran Rp 10.000
              </p>
            </div>

            {/* Pilihan Metode Pembayaran */}
            <div>
              <label
                htmlFor="method"
                className="block text-sm font-medium text-gray-700"
              >
                Metode Pembayaran
              </label>
              <div className="mt-1">
                <select
                  id="method"
                  name="method"
                  value={method}
                  onChange={(e) => setMethod(e.target.value)}
                  className="block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md border"
                >
                  <option value="BCA">BCA Virtual Account</option>
                  <option value="MANDIRI">Mandiri Virtual Account</option>
                  <option value="OVO">OVO / E-Wallet</option>
                  <option value="QRIS">QRIS</option>
                </select>
              </div>
            </div>

            {/* Pesan Error */}
            {errorMessage && (
              <div className="bg-red-50 border-l-4 border-red-400 p-4">
                <div className="flex">
                  <div className="ml-3">
                    <p className="text-sm text-red-700">{errorMessage}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Tombol Submit */}
            <div>
              <button
                type="submit"
                disabled={isLoading || amount === "" || amount < 10000}
                className={`w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white transition-colors duration-200 ${
                  isLoading || amount === "" || amount < 10000
                    ? "bg-blue-300 cursor-not-allowed"
                    : "bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                }`}
              >
                {isLoading ? (
                  <span className="flex items-center">
                    <svg
                      className="animate-spin -ml-1 mr-3 h-5 w-5 text-white"
                      xmlns="http://www.w3.org/2000/svg"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      ></circle>
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    Memproses...
                  </span>
                ) : (
                  "Bayar Sekarang"
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
